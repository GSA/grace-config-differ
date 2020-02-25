.PHONY: test_lambda release_lambda clean
test_lambda:
	make -C handler test

release_lambda:
	make -C handler release

clean:
	make -C handler clean
