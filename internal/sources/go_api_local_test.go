package sources_test

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/internal/sources"
)

func TestGuardedGoAPIWorkspaceBuildsPublishesAndRunsManuallyAndOnSchedule(t *testing.T) {
	testGuardedAPIWorkspace(t, "go")
}

func guardedGoZIP(t *testing.T, draft sources.PlaywrightDraft, origin string) []byte {
	t.Helper()
	files := map[string]string{"go.mod": "module teststudio.local/guarded-api\n\ngo 1.26.0\n", "checks/api_test.go": strings.ReplaceAll(guardedGoAPICode, "__DIRECT_ORIGIN__", origin)}
	for _, file := range draft.Files {
		if file.Path == draft.ManifestPath || file.Path == "catalog.json" {
			files[file.Path] = file.Content
		}
	}
	var data bytes.Buffer
	writer := zip.NewWriter(&data)
	for name, body := range files {
		file, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return data.Bytes()
}

const guardedGoAPICode = `package checks
import (
 "bufio"
 "encoding/json"
 "fmt"
 "io"
 "net"
 "net/http"
 "net/url"
 "os"
 "strings"
 "testing"
 "time"
)
func TestGuardedAPIRequests(t *testing.T) {
 base:=os.Getenv("TESTSTUDIO_API_BASE_URL")
 if base=="" || os.Getenv("TESTSTUDIO_BASE_URL")!=base || os.Getenv("HTTP_PROXY")!="" {t.Fatal("Guarded Go request environment is missing")}
 client:=&http.Client{Timeout:3*time.Second}
 defer client.CloseIdleConnections()
 request:=func(method,path,body string,headers map[string]string)(int,string,http.Header){
  t.Helper(); req,err:=http.NewRequest(method,base+path,strings.NewReader(body));if err!=nil {t.Fatal(err)}
  for name,value:=range headers {req.Header.Set(name,value)}
  response,err:=client.Do(req);if err!=nil {t.Fatal(err)};defer response.Body.Close()
  data,err:=io.ReadAll(response.Body);if err!=nil {t.Fatal(err)}
  return response.StatusCode,string(data),response.Header
 }
 t.Run("Read the permitted endpoint with injected authentication",func(t *testing.T){
  code,body,_:=request("GET","/health/","",nil);if code!=200 {t.Fatalf("Expected authorized status, received %d",code)}
  var value map[string]any;if json.Unmarshal([]byte(body),&value)!=nil || value["status"]!="up" || value["echo"]!="[REDACTED]" {t.Fatal("Response authentication or credential reflection boundary failed")}
  t.Logf("Response diagnostics: %s",body)
 })
 t.Run("Reject unreviewed methods paths overrides and oversized bodies",func(t *testing.T){
  for _,value:=range []struct{method,path,body string;code int;headers map[string]string}{
   {"DELETE","/health","",403,nil},{"GET","/outside","",403,nil},{"GET","/health/%2e%2e/private","",403,nil},
   {"GET","/health","",403,map[string]string{"X-HTTP-Method-Override":"DELETE"}},{"POST","/search","123456789",413,nil},
  } {if code,_,_:=request(value.method,value.path,value.body,value.headers);code!=value.code {t.Fatalf("Denied request returned %d",code)}}
 })
 t.Run("Perform the reviewed read-only POST",func(t *testing.T){if code,_,_:=request("POST","/search","owned",nil);code!=200 {t.Fatalf("Read-only POST returned %d",code)}})
 t.Run("Retain redirect status without following it",func(t *testing.T){
  code,body,headers:=request("GET","/redirect","",nil)
  if code!=302 || headers.Get("Location")!="" || strings.Contains(body,"private-route") {t.Fatal("Redirect policy failed")}
 })
 t.Run("Reject direct network and CONNECT bypasses",func(t *testing.T){
  direct,err:=net.DialTimeout("tcp",strings.TrimPrefix("__DIRECT_ORIGIN__","http://"),250*time.Millisecond)
  if err==nil {_ = direct.Close();t.Fatal("API worker reached its target outside the gateway")}
  address,err:=url.Parse(base);if err!=nil {t.Fatal(err)}
  connection,err:=net.DialTimeout("tcp",address.Host,time.Second);if err!=nil {t.Fatal(err)};defer connection.Close();_ = connection.SetDeadline(time.Now().Add(time.Second))
  _,err=fmt.Fprint(connection,"CONNECT blocked.invalid:443 HTTP/1.1\r\nHost: blocked.invalid:443\r\n\r\n");if err!=nil {t.Fatal(err)}
  response,err:=http.ReadResponse(bufio.NewReader(connection),nil);if err!=nil {t.Fatal(err)};defer response.Body.Close()
  if response.StatusCode!=403 {t.Fatal("CONNECT escaped the reviewed request transport")}
 })
 t.Run("Enforce the attempt request budget",func(t *testing.T){if code,_,_:=request("GET","/health","",nil);code!=429 {t.Fatalf("Request budget returned %d",code)}})
}
`
