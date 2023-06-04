package utils

import "net/url"

// ParseReqHandler достает из строки имя обработчика команды telegram
func ParseReqHandler(reqURL string) string {
	reqUrl, err := url.Parse(reqURL)
	if err != nil {
		return ""
	}
	return reqUrl.Opaque
}

// ParseReqParams достает из строки параметры запроса
func ParseReqParams(reqURL string) map[string][]string {
	empty := make(map[string][]string)
	reqUrl, err := url.Parse(reqURL)
	if err != nil {
		return empty
	}
	params, err := url.ParseQuery(reqUrl.RawQuery)
	if err != nil {
		return empty
	}
	return params
}
