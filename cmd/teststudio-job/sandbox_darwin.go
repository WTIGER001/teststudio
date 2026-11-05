package main

import "errors"

func sandboxIdentity() error { return errors.New("sandbox command supervision requires Linux") }
