package main

func fail_on_err(err error) {
	if err != nil {
		panic(err)
	}
}
