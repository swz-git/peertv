package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var searxEngines []string

type SearchResult struct {
	URL           string   `json:"url"`
	Title         string   `json:"title"`
	Seed          string   `json:"seed"`
	Leech         string   `json:"leech"`
	MagnetLink    string   `json:"magnetlink,omitempty"`
	Template      string   `json:"template"`
	PublishedDate string   `json:"publishedDate,omitempty"`
	FileSize      int      `json:"filesize"`
	Engine        string   `json:"engine"`
	ParsedURL     []string `json:"parsed_url"`
	Engines       []string `json:"engines"`
	Positions     []int    `json:"positions"`
	Score         float64  `json:"score"`
	Category      string   `json:"category"`
	PrettyURL     string   `json:"pretty_url"`
	PubDate       string   `json:"pubdate,omitempty"`
	OpenGroup     bool     `json:"open_group,omitempty"`
	CloseGroup    bool     `json:"close_group,omitempty"`
}

type SearxResponse struct {
	Query               string           `json:"query"`
	NumberOfResults     int              `json:"number_of_results"`
	Results             SearchResultList `json:"results"`
	Answers             []interface{}    `json:"answers"`
	Corrections         []interface{}    `json:"corrections"`
	Infoboxes           []interface{}    `json:"infoboxes"`
	Suggestions         []interface{}    `json:"suggestions"`
	UnresponsiveEngines [][]string       `json:"unresponsive_engines"`
}

type SearchResultList []SearchResult

func (e SearchResultList) Len() int {
	return len(e)
}

func (e SearchResultList) Less(i, j int) bool {
	one, err := strconv.Atoi(e[i].Seed)
	two, err := strconv.Atoi(e[j].Seed)
	if err != nil {
		return false
	}
	return one > two
}

func (e SearchResultList) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

type LoggableSearchResult struct {
	URL        string  `json:"url"`
	Title      string  `json:"title"`
	Seed       string  `json:"seed"`
	Leech      string  `json:"leech"`
	MagnetLink string  `json:"magnetlink,omitempty"`
	FileSize   int     `json:"filesize"`
	Engine     string  `json:"engine"`
	Score      float64 `json:"score"`
}

func mapSearchResults(sr []SearchResult) []LoggableSearchResult {
	var results []LoggableSearchResult
	for _, r := range sr {
		if r.MagnetLink != "" {
			results = append(results, LoggableSearchResult{
				URL:        r.URL,
				Title:      r.Title,
				Seed:       r.Seed,
				Leech:      r.Leech,
				MagnetLink: r.MagnetLink,
				FileSize:   r.FileSize,
				Engine:     r.Engine,
				Score:      r.Score,
			})
		}
	}
	return results
}

// searchCmd represents the search command
var searchCmd = &cobra.Command{
	Use:   "search [search term]",
	Short: "Search for a magnet link using searx",
	Run: func(cmd *cobra.Command, args []string) {
		searchTerm := args[0]

		searxInstance := cmd.Flag("searx-instance").Value.String()

		u, err := url.ParseRequestURI(searxInstance)
		if err != nil {
			panic(err)
		}

		searxInstance = u.String()

		if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			panic("invalid searx instance")
		}

		fmt.Errorf("Searching for \"%s\" using %s (%s)\n", searchTerm, strings.Join(searxEngines, ","), searxInstance)

		params := url.Values{}
		params.Add("q", searchTerm)
		params.Add("engines", cmd.Flag("searx-engines").Value.String())
		params.Add("format", "json")
		params.Add("categories", "files")

		u.RawQuery = params.Encode()

		searxInstance = u.String()

		res, err := http.Get(searxInstance)

		if err != nil {
			panic(err)
		}

		defer res.Body.Close()
		bodyBytes, _ := ioutil.ReadAll(res.Body)

		// bodyString := string(bodyBytes)
		// fmt.Println("API Response as String:\n" + bodyString)

		var searxResponse SearxResponse
		json.Unmarshal(bodyBytes, &searxResponse)

		if cmd.Flag("json").Value.String() == "true" {
			loggableSearchResult, err := json.Marshal(mapSearchResults(searxResponse.Results))
			if err != nil {
				panic(err)
			}
			fmt.Printf("%s", loggableSearchResult)
		} else {
			results := searxResponse.Results
			sort.Sort(results)
			for _, sr := range results {
				fmt.Printf("%s\n", sr.MagnetLink)
			}
		}

	},
}

func init() {
	rootCmd.AddCommand(searchCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// searchCmd.PersistentFlags().String("searx-instance", "i", "SearX instance to use")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	searchCmd.Flags().StringP("searx-instance", "i", "", "SearX instance to use")
	cobra.MarkFlagRequired(searchCmd.Flags(), "searx-instance")

	searchCmd.Flags().StringSliceVarP(&searxEngines, "searx-engines", "e", []string{"1337x", "nyaa", "kickass", "piratebay"}, "SearX engines to use")

	searchCmd.Flags().Bool("json", false, "Output in JSON format")
}
