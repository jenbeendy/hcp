package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
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
	TournamentName    string   `json:"tournamentName"`
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

type TournamentCategory struct {
	TournamentCategoryID int64  `json:"tournamentCategoryId"`
	Name                 string `json:"name"`
	PlayingSystemName    string `json:"playingSystemName"`
	Order                int    `json:"order"`
	Main                 bool   `json:"main"`
	HcpUse               bool   `json:"hcpUse"`
	HcpCorrection        bool   `json:"hcpCorrection"`
}

type TournamentEntry struct {
	GolferID      int64   `json:"golferId"`
	GolferName    string  `json:"golferName"`
	ClubShortName string  `json:"clubShortName"`
	HcpText       string  `json:"hcpText"`
	HcpNumber     float64 `json:"hcpNumber"`
	Categories    []struct {
		CategoryID int64 `json:"categoryId"`
	} `json:"categories"`
}

type Client struct {
	token string
	http  *http.Client

	mu       sync.Mutex
	lastCall time.Time
}

// minCallGap keeps API calls at least this far apart so the servers
// are not overloaded.
const minCallGap = 50 * time.Millisecond

func NewClient(token string) *Client {
	return &Client{
		token: token,
		http:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) throttle() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if wait := minCallGap - time.Since(c.lastCall); wait > 0 {
		time.Sleep(wait)
	}
	c.lastCall = time.Now()
}

func (c *Client) do(ctx context.Context, url string) (*http.Response, error) {
	c.throttle()
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

func (c *Client) GetTournamentCategories(ctx context.Context, tournamentID string) ([]TournamentCategory, error) {
	url := fmt.Sprintf("https://api.cgf.cz/api/v1/tournament/%s/category", tournamentID)
	resp, err := c.do(ctx, url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API %s", resp.Status)
	}
	var result []TournamentCategory
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GetTournamentEntries(ctx context.Context, tournamentID string) ([]TournamentEntry, error) {
	url := fmt.Sprintf("https://api.cgf.cz/api/v1/tournament/%s/entry", tournamentID)
	resp, err := c.do(ctx, url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API %s", resp.Status)
	}
	var result []TournamentEntry
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
