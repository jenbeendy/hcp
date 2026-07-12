package handlers

import (
	"bytes"
	"encoding/json"
	"html/template"
	"os"
	"testing"

	"github.com/hcp/golf/api"
)

func TestScorecardRender(t *testing.T) {
	data, err := os.ReadFile("testdata/rounddetail.json")
	if err != nil {
		t.Fatal(err)
	}
	var details []api.RoundDetail
	if err := json.Unmarshal(data, &details); err != nil {
		t.Fatal(err)
	}
	page := ScorecardPage{}
	for _, d := range details {
		page.Rounds = append(page.Rounds, convertScorecard(d, len(details) > 1))
	}
	tmpl := template.Must(template.ParseFiles("../templates/scorecard.html"))
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "scorecard", page); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"Testville", "OUT", "IN", "score-birdie", "score-double", ">81<", ">35<", ">42<", ">39<"} {
		if !bytes.Contains(buf.Bytes(), []byte(want)) {
			t.Errorf("output missing %q", want)
		}
	}
	t.Log(out)
}
