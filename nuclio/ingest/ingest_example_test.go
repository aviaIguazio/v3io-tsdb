package ingest

import (
	"fmt"
	"github.com/nuclio/nuclio-test-go"
	"os"
	"testing"
	"time"
)

func TestIngest(t *testing.T) {
	data := nutest.DataBind{Name: "db0", Url: os.Getenv("V3IO_URL"), Container: "1"}
	tc, err := nutest.NewTestContext(Handler, true, &data)
	if err != nil {
		t.Fatal(err)
	}

	err = tc.InitContext(InitContext)
	if err != nil {
		t.Fatal(err)
	}
	testEvent := nutest.TestEvent{
		Body: []byte(pushEvent),
	}
	resp, err := tc.Invoke(&testEvent)
	tc.Logger.InfoWith("Run complete", "resp", resp, "err", err)
	resp, err = tc.Invoke(&testEvent)
	time.Sleep(time.Second * 1)
	tc.Logger.InfoWith("Run complete", "resp", resp, "err", err)
	fmt.Println(resp)

	time.Sleep(time.Second * 10)
}
