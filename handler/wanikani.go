package handler

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo"
	"github.com/patrickmn/go-cache"
)

//
var waniKaniAPIURL string = "https://api.wanikani.com/v2"
var statisticsEndpoint string = "/review_statistics"
var subjectsEndpoint string = "/subjects"
var criticalItemPercentage string = "75"

type intSlice []int

func (l *intSlice) pop() int {
	length := len(*l)
	lastElement := (*l)[length-1]
	*l = (*l)[:length-1]
	return lastElement
}

type reviewStatistics struct {
	TotalCount int `json:"total_count"`
	Data       []struct {
		Data struct {
			SubjectID         int    `json:"subject_id"`
			SubjectType       string `json:"subject_type"`
			PercentageCorrect int    `json:"percentage_correct"`
		} `json:"data"`
	} `json:"data"`
}

func getReviewStatistics(c *http.Client, token string) reviewStatistics {
	url := waniKaniAPIURL + statisticsEndpoint
	req, err := http.NewRequest("GET", url, nil)
	bearer := "Bearer " + token
	req.Header.Add("Authorization", bearer)

	q := req.URL.Query()
	q.Add("percentages_less_than", criticalItemPercentage)
	req.URL.RawQuery = q.Encode()

	resp, err := c.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	bodyBytes, _ := ioutil.ReadAll(resp.Body)

	var statistics reviewStatistics
	json.Unmarshal(bodyBytes, &statistics)

	return statistics
}

func getCriticalItemIds(c *http.Client, token string) []int {
	criticalItems := getReviewStatistics(c, token)
	itemIds := []int{}
	for _, item := range criticalItems.Data {
		itemIds = append(itemIds, item.Data.SubjectID)
	}
	return itemIds
}

// SubjectData - contains the struct needed to parse wanikani subject JSON
// data
type SubjectData struct {
	ID     int    `json:"id"`
	Object string `json:"object"`
	URL    string `json:"url"`
	Data   struct {
		CreatedAt   time.Time   `json:"created_at"`
		Level       int         `json:"level"`
		Slug        string      `json:"slug"`
		HiddenAt    interface{} `json:"hidden_at"`
		DocumentURL string      `json:"document_url"`
		Characters  string      `json:"characters"`
		Meanings    []struct {
			Meaning        string `json:"meaning"`
			Primary        bool   `json:"primary"`
			AcceptedAnswer bool   `json:"accepted_answer"`
		} `json:"meanings"`
		AuxiliaryMeanings []interface{} `json:"auxiliary_meanings"`
		Readings          []struct {
			Type           string `json:"type"`
			Primary        bool   `json:"primary"`
			Reading        string `json:"reading"`
			AcceptedAnswer bool   `json:"accepted_answer"`
		} `json:"readings"`
		ComponentSubjectIds       []int  `json:"component_subject_ids"`
		AmalgamationSubjectIds    []int  `json:"amalgamation_subject_ids"`
		VisuallySimilarSubjectIds []int  `json:"visually_similar_subject_ids"`
		MeaningMnemonic           string `json:"meaning_mnemonic"`
		MeaningHint               string `json:"meaning_hint"`
		ReadingMnemonic           string `json:"reading_mnemonic"`
		ReadingHint               string `json:"reading_hint"`
	} `json:"data,omitempty"`
}

type Subjects struct {
	Data []SubjectData `json:"data"`
}

func getSubjects(c *http.Client, ids []int, token string) Subjects {
	url := waniKaniAPIURL + subjectsEndpoint
	req, err := http.NewRequest("GET", url, nil)
	bearer := "Bearer " + token
	req.Header.Add("Authorization", bearer)

	formattedIds := intSliceToString(ids, ",")
	q := req.URL.Query()
	q.Add("ids", formattedIds)
	req.URL.RawQuery = q.Encode()

	resp, err := c.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := ioutil.ReadAll(resp.Body)

	var criticalSubjects Subjects
	json.Unmarshal(bodyBytes, &criticalSubjects)

	return criticalSubjects
}

// GetCriticalSubjects - Retrieves the critical review items from a WaniKani
// user based on the token given. Retruns a struct 'Subjects' which contains
// the critical items and their details.
func (h *Handler) GetCriticalSubjects(c echo.Context) error {
	token := c.QueryParam("APIToken")
	ids := getCriticalItemIds(h.Client, token)

	cachedSubjects := Subjects{}
	uncachedIds := []int{}
	for _, id := range ids {
		key := "subjects:" + strconv.Itoa(id)
		subject, found := h.GoCache.Get(key)
		if found {
			cachedSubjects.Data = append(cachedSubjects.Data, subject.(SubjectData))
		} else {
			uncachedIds = append(uncachedIds, id)
		}
	}

	if len(uncachedIds) > 0 {
		uncachedSubjects := getSubjects(h.Client, uncachedIds, token)
		for _, subject := range uncachedSubjects.Data {
			key := "subjects:" + strconv.Itoa(subject.ID)
			h.GoCache.Set(key, subject, cache.NoExpiration)
		}
		cachedSubjects.Data = append(cachedSubjects.Data, uncachedSubjects.Data...)
	}

	return c.JSON(http.StatusOK, cachedSubjects)
}

// Utility method for converting a slice of integers to a string separated
// by a delimiter
func intSliceToString(s []int, delimiter string) string {
	stringSlice := []string{}

	for i := range s {
		text := strconv.Itoa(s[i])
		stringSlice = append(stringSlice, text)
	}

	return strings.Join(stringSlice, delimiter)
}
