/*
history:
2020/4/10 v1
2020/6/26 config file support

GoFmt GoBuildNull GoBuild GoRelease
*/

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

var (
	ConfigPath = "$HOME/config/youtube.keys"
	YtKey      string

	HttpClient = &http.Client{}

	YtPlaylistRe       *regexp.Regexp
	YtPlaylistReString = `youtube.com/.*[?&]list=([0-9A-Za-z_-]+)$`

	YtMaxResults = 50
	TitleMaxLen  = 50
)

type YtPlaylistItemSnippet struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	PublishedAt string `json:"publishedAt"`
	Thumbnails  struct {
		Medium struct {
			Url string `json:"url"`
		} `json:"medium"`
		High struct {
			Url string `json:"url"`
		} `json:"high"`
		Standard struct {
			Url string `json:"url"`
		} `json:"standard"`
		MaxRes struct {
			Url string `json:"url"`
		} `json:"maxres"`
	} `json:"thumbnails"`
	Position   int64 `json:"position"`
	ResourceId struct {
		VideoId string `json:"videoId"`
	} `json:"resourceId"`
}

type YtPlaylistItem struct {
	Snippet YtPlaylistItemSnippet `json:"snippet"`
}

type YtPlaylistItems struct {
	NextPageToken string `json:"nextPageToken"`
	PageInfo      struct {
		TotalResults   int64 `json:"totalResults"`
		ResultsPerPage int64 `json:"resultsPerPage"`
	} `json:"pageInfo"`
	Items []YtPlaylistItem
}

func init() {
	YtPlaylistRe = regexp.MustCompile(YtPlaylistReString)

	if os.Getenv("YtKey") != "" {
		YtKey = os.Getenv("YtKey")
	}

	if YtKey == "" {
		ConfigPath = os.ExpandEnv(ConfigPath)
		configBb, err := ioutil.ReadFile(ConfigPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "reading %s: %v", ConfigPath, err)
		}

		for _, configLine := range strings.Split(string(configBb), "\n") {
			configLine = strings.TrimSpace(configLine)
			if configLine == "" || strings.HasPrefix(configLine, "#") {
				continue
			}

			kv := strings.Split(configLine, "=")
			if len(kv) != 2 || strings.TrimSpace(kv[0]) == "" {
				fmt.Fprintf(os.Stderr, "invalid %s config line: %s\n", ConfigPath, configLine)
				continue
			}

			k, v := strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])
			v = strings.Trim(v, `'"`)
			if k == "YtKey" {
				YtKey = v
			}
		}
	}

	if YtKey == "" {
		fmt.Fprintln(os.Stderr, "No YtKey provided")
		os.Exit(1)
	}
}

func getJson(url string, target interface{}) error {
	r, err := HttpClient.Get(url)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	return json.NewDecoder(r.Body).Decode(target)
}

func safestring(s string) (t string) {
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			r = '.'
		}
		t = t + string(r)
	}

	if len([]rune(t)) > TitleMaxLen {
		t = string([]rune(t)[:TitleMaxLen])
	}

	return t
}

func main() {
	var err error
	var ytplid string

	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: tube.ls youtube.playlist.id\n")
		os.Exit(1)
	}
	ytplid = os.Args[1]

	mm := YtPlaylistRe.FindStringSubmatch(ytplid)
	if len(mm) > 1 {
		ytplid = mm[1]
	}

	var videos []YtPlaylistItemSnippet
	nextPageToken := ""

	for nextPageToken != "" || len(videos) == 0 {
		var PlaylistItemsUrl = fmt.Sprintf("https://www.googleapis.com/youtube/v3/playlistItems?maxResults=%d&part=snippet&playlistId=%s&key=%s&pageToken=%s", YtMaxResults, ytplid, YtKey, nextPageToken)

		var playlistItems YtPlaylistItems
		err = getJson(PlaylistItemsUrl, &playlistItems)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get playlist items: %v", err)
			os.Exit(1)
		}

		if playlistItems.NextPageToken != nextPageToken {
			nextPageToken = playlistItems.NextPageToken
		} else {
			nextPageToken = ""
		}

		for _, i := range playlistItems.Items {
			videos = append(videos, i.Snippet)
		}
	}

	sort.Slice(videos, func(i, j int) bool { return videos[i].PublishedAt < videos[j].PublishedAt })
	counterlen := int(math.Log10(float64(len(videos)))) + 1
	numfmt := "%0" + strconv.Itoa(counterlen) + "d"

	for vidnum, vid := range videos {
		ytid := vid.ResourceId.VideoId
		fmt.Printf(
			//"https://youtu.be/%s "+numfmt+".%s\n",
			//ytid, vidnum+1, safestring(vid.Title),
			"https://youtu.be/%s "+numfmt+".\n",
			ytid, vidnum+1,
		)
	}
}
