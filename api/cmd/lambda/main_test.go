package main

import (
	"encoding/json"
	"testing"

	"counter-drop/api/internal/store/ddbstore"

	"github.com/aws/aws-lambda-go/events"
)

// A DynamoDB stream record as Lambda delivers it must decode into the same events as the store's own images.
func TestStreamRecordToEvents(t *testing.T) {
	raw := `{"Records":[{"eventID":"1","dynamodb":{
	  "OldImage":{"PK":{"S":"J#job_1"},"SK":{"S":"JOB"},"Ver":{"N":"3"},
	    "Job":{"M":{"ID":{"S":"job_1"},"ShopID":{"S":"shop_1"},"State":{"S":"uploading"},"Token":{"S":""},"FilesDeletedAt":{"NULL":true}}},
	    "Files":{"L":[{"M":{"Seq":{"N":"0"},"F":{"M":{"ID":{"S":"file_1"},"Downloads":{"N":"0"},"DownloadedBy":{"S":""}}}}}]}},
	  "NewImage":{"PK":{"S":"J#job_1"},"SK":{"S":"JOB"},"Ver":{"N":"4"},
	    "Job":{"M":{"ID":{"S":"job_1"},"ShopID":{"S":"shop_1"},"State":{"S":"queued"},"Token":{"S":"A-01"},"FilesDeletedAt":{"NULL":true}}},
	    "Files":{"L":[{"M":{"Seq":{"N":"0"},"F":{"M":{"ID":{"S":"file_1"},"Downloads":{"N":"1"},"DownloadedBy":{"S":"Kavita"}}}}}]}}
	}}]}`
	var ev events.DynamoDBEvent
	if err := json.Unmarshal([]byte(raw), &ev); err != nil {
		t.Fatal(err)
	}
	rec := ev.Records[0]
	msgs, err := ddbstore.ChangeEvents(toAV(rec.Change.OldImage), toAV(rec.Change.NewImage))
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, m := range msgs {
		got[m.Event.Type] = m.Topic
	}
	want := map[string]string{"job.updated": "job:job_1", "file.downloaded": "job:job_1", "queue.changed": "shop:shop_1"}
	for k, v := range want {
		if got[k] != v {
			t.Fatalf("event %s → %q, want %q (all: %v)", k, got[k], v, got)
		}
	}
}
