package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/nsf/jsondiff"
)

const (
	blankRow    = "<tr><td class=\"blank\" colspan=4>&nbsp;</td></tr>\n"
	blankCol    = "<td class=\"blank\">&nbsp;</td>"
	headerRow   = "<tr><td class=\"blank\">&nbsp;</td><th>Property</th><th>Previous</th><th>Current</th></tr>\n"
	indent      = "&nbsp;&nbsp;"
	shortFldLen = 40  // Reasonable field to compare side-by-side
	longFldLen  = 400 // Resonable field to compare as diff
)

// parseItemsToHTML ... generic parsing of configservice.ConfigurationItems into html
func parseItemsToHTML(items []map[string]interface{}) (string, error) {
	str := ""

	for _, i := range items {
		if i["ResourceName"] == nil || i["ResourceName"] == "" {
			str = fmt.Sprintf("%s%s<tr><td class=\"resource\" colspan=2>%s</td><td class=\"resource\" colspan=2>%s</td></tr>\n",
				str, blankRow, i["ResourceId"], i["ResourceType"])
		} else {
			str = fmt.Sprintf("%s%s<tr><td class=\"resource\" colspan=2>%s</td><td class=\"resource\" colspan=2>%s</td></tr>\n",
				str, blankRow, i["ResourceName"], i["ResourceType"])
		}

		if val, ok := i["diffs"]; ok {
			// There was a snapshot of this item
			s, err := diffsToHTML(val.(map[string]interface{}), i, "")
			if err != nil {
				return s, err
			}

			str += s
		} else {
			// There was no snapshot of this item, so assume it is new
			endRow := "</td></tr>\n"
			str = str[:len(str)-len(endRow)] + " (New Item)" + endRow
			slice, err := json.MarshalIndent(i, "", indent)
			if err != nil {
				return "", err
			}
			s := string(slice)
			re := regexp.MustCompile("\"([\\w]+)\":")
			s = re.ReplaceAllStringFunc(s, addStrong)
			s = strings.Replace(s, "\n", "<br />\n", -1)
			str += "<tr><td>&nbsp</td><td colspan=3>" + s + endRow
		}
	}

	return str, nil
}

func addStrong(s string) string {
	return fmt.Sprintf("<strong>%s</strong>", s)
}

func diffsToHTML(diffs, item map[string]interface{}, group string) (string, error) {
	var str string

	if group == "" {
		str = blankRow + headerRow
	} else {
		str = fmt.Sprintf("<tr>%s<th class=\"group\" colspan=\"3\">%s</th></tr>\n", blankCol, group)
	}

	// Process all top level attributes
	for key, value := range diffs {
		switch t := value.(type) {
		case map[string]interface{}:
		default:
			s, err := trDiff(key, t, item[key])
			if err != nil {
				return "", err
			}

			str += s
		}
	}

	// Process recursive attributes
	for key, value := range diffs {
		switch t := value.(type) {
		case map[string]interface{}:
			if val, ok := t["diffs"]; ok {
				s, err := diffsToHTML(val.(map[string]interface{}), item[key].(map[string]interface{}), key)
				if err != nil {
					return "", err
				}

				str += s

				break
			} else {
				s, err := trDiff(key, t, item[key])
				if err != nil {
					return "", err
				}

				str += s
			}
		default:
		}
	}

	return str, nil
}

func trDiff(k string, old, newer interface{}) (s string, err error) {
	a, b, err := myMarshal(old, newer)
	if err != nil {
		return "", err
	}

	if len(a) <= shortFldLen && len(b) <= shortFldLen {
		s = fmt.Sprintf("<tr>%s<th>%s</th><td>%s</td><td>%s</td></tr>\n", blankCol, k, a, b)
	} else if len(a) <= longFldLen && len(b) <= longFldLen {
		s = fmt.Sprintf("<tr>%s<th>%s</th><td colspan=2>%s</td></tr>\n", blankCol, k, ppDiff(a, b))
	} else {
		s = fmt.Sprintf("<tr>%s<th>%s</th><td colspan=2 align=\"center\"><em>long output suppressed</em></td></tr>\n", blankCol, k)
	}

	return s, nil
}

func myMarshal(old, newer interface{}) (a, b []byte, err error) {
	a, err = json.Marshal(old)
	if err != nil {
		return nil, nil, err
	}

	if string(a) == nullStr {
		a = []byte("[]")
	}

	b, err = json.Marshal(newer)
	if err != nil {
		return nil, nil, err
	}

	return a, b, nil
}

func ppDiff(a, b []byte) string {
	opts := jsondiff.DefaultHTMLOptions()
	opts.Indent = indent
	_, s := jsondiff.Compare(a, b, &opts)
	s = strings.Replace(s, "\n", "<br />\n", -1)

	return s
}
