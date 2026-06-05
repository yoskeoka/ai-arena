package service

import "net/url"

func isLocalPath(locator string) bool {
	parsed, err := url.Parse(locator)
	if err != nil {
		return true
	}
	return parsed.Scheme == "" || parsed.Scheme == "file"
}
