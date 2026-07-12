package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type HCPRecord struct {
	Date              string   `json:"date"`
	Par               int      `json:"par"`
	CR                float64  `json:"cr"`
	SR                int      `json:"sr"`
	Strokes           *int     `json:"strokes"`
	Points            int      `json:"points"`
	UHV               int      `json:"uhv"`
	PCC               int      `json:"pcc"`
	SU                int      `json:"su"`
	PO                float64  `json:"po"`
	WhsHI             string   `json:"whsHI"`
	Indicators        []string `json:"indicators"`
	OutstandingResult bool     `json:"outstandingResult"`
	TournamentID      int64    `json:"tournamentId"`
	TournamentRoundID int64    `json:"tournamentRoundId"`
}

type RoundHole struct {
	HoleIndex       int    `json:"holeIndex"`
	Par             int    `json:"par"`
	Length          int    `json:"length"`
	HcpIndex        int    `json:"hcpIndex"`
	Strokes         string `json:"strokes"`
	HcpStrokes      int    `json:"hcpStrokes"`
	StablefordNetto int    `json:"stablefordNetto"`
	ResType         string `json:"resType"`
}

type RoundDetail struct {
	GolferName      string      `json:"golferName"`
	CourseName      string      `json:"courseName"`
	StipRoundName   string      `json:"stipRoundName"`
	TeeColorCode    string      `json:"teeColorCode"`
	RoundDate       string      `json:"roundDate"`
	Par             int         `json:"par"`
	CR              float64     `json:"cr"`
	SR              int         `json:"sr"`
	PCC             int         `json:"pcc"`
	HcpBefore       string      `json:"hcpBefore"`
	HcpAfter        string      `json:"hcpAfter"`
	PlayingHcp      int         `json:"playingHcp"`
	HolesGrouping   int         `json:"holesGrouping"`
	Holes           []RoundHole `json:"holes"`
	RoundID         int64       `json:"roundId"`
	RoundIndex      int         `json:"roundIndex"`
	Strokes         int         `json:"strokes"`
	StablefordNetto int         `json:"stablefordNetto"`
}

type HCPHistoryResponse struct {
	FullName     string      `json:"fullName"`
	HomeClub     string      `json:"homeClub"`
	MemberNumber string      `json:"memberNumber"`
	CurrentHI    string      `json:"currentHI"`
	HcpRecords   []HCPRecord `json:"hcpRecords"`
}

type Registration struct {
	TournamentID       int64  `json:"tournamentId"`
	Name               string `json:"name"`
	ShortName          string `json:"shortName"`
	DateActionFrom     string `json:"dateActionFrom"`
	DateActionTo       string `json:"dateActionTo"`
	CourseName         string `json:"courseName"`
	NumberOfRounds     int    `json:"numberOfRounds"`
	DateRegistrationTo string `json:"dateRegistrationTo"`
	State              string `json:"state"`
	Fees               string `json:"fees"`
	IsRegistered       bool   `json:"isRegistered"`
}

type Client struct {
	token string
	http  *http.Client
}

func NewClient(token string) *Client {
	return &Client{
		token: token,
		http:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) do(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "ktor-client")
	return c.http.Do(req)
}

func (c *Client) GetHCPHistory(ctx context.Context, golferID string) (*HCPHistoryResponse, error) {
	url := fmt.Sprintf("https://api.cgf.cz/api/v1/golfer/%s/hcp-history", golferID)
	resp, err := c.do(ctx, url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API %s", resp.Status)
	}
	var result HCPHistoryResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetRoundDetail(ctx context.Context, tournamentID, golferID string) ([]RoundDetail, error) {
	url := fmt.Sprintf("https://api.cgf.cz/api/v1/tournament/%s/result/golfer/%s/detail", tournamentID, golferID)
	resp, err := c.do(ctx, url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API %s", resp.Status)
	}
	var result []RoundDetail
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GetRegistrations(ctx context.Context, golferID, from, to string) ([]Registration, error) {
	url := fmt.Sprintf(
		"https://api.cgf.cz/api/v1/tournament/by-golfer?golferId=%s&golferIsParticipant=true&dateFrom=%s&dateTo=%s",
		golferID, from, to,
	)
	resp, err := c.do(ctx, url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API %s", resp.Status)
	}
	var result []Registration
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}
