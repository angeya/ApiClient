package fetch

import (
	"testing"
)

func TestParseSample(t *testing.T) {
	code := "fetch(`http://10.44.2.115:9090/mainApi/syncplant-system-resource/oss/getAllFile?_menuId=857045958305047064`, {\n" +
		"  \"headers\": {\n" +
		"    \"accept\": \"application/json, text/plain, */*\",\n" +
		"    \"accept-language\": \"zh\",\n" +
		"    \"authorization\": \"EVCv3/1p6bXTcOWaelB/mhz2+Ejf4isAEJORGyAj1szluoWdNluakFg8vSHvyYwY\",\n" +
		"    \"cache-control\": \"no-cache\",\n" +
		"    \"pragma\": \"no-cache\",\n" +
		"    \"sciyon-auth\": \"bearer c537cfb9-4b53-48f5-b3bd-89d5d9be2fc8\",\n" +
		"    \"sciyon-permission\": \"e30=\",\n" +
		"    \"sciyonrequestkey\": \"d564key@cee438b9f6a506c16888247ea807\",\n" +
		"    \"tenant-id\": \"sciyon\"\n" +
		"  },\n" +
		"  \"referrer\": `http://10.44.2.115:9090/pf/other-setting.html`,\n" +
		"  \"body\": null,\n" +
		"  \"method\": \"GET\",\n" +
		"  \"mode\": \"cors\",\n" +
		"  \"credentials\": \"include\"\n" +
		"});"

	result, err := Parse(code)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if result.URL != "http://10.44.2.115:9090/mainApi/syncplant-system-resource/oss/getAllFile?_menuId=857045958305047064" {
		t.Fatalf("unexpected URL: %s", result.URL)
	}
	if result.Method != "GET" {
		t.Fatalf("expected GET, got %s", result.Method)
	}
	if len(result.Headers) != 9 {
		t.Fatalf("expected 9 headers, got %d", len(result.Headers))
	}
	if len(result.Query) != 1 || result.Query[0].Key != "_menuId" {
		t.Fatalf("expected 1 query param _menuId, got %v", result.Query)
	}
	if result.Body != "" {
		t.Fatalf("expected empty body, got %q", result.Body)
	}
}
