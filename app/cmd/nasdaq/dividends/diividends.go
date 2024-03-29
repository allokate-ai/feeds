package dividends

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/allokate-ai/events/app/pkg/client"
	"github.com/spf13/cobra"
)

func ParseDate(text string) *time.Time {
	date, err := time.Parse("01/02/2006", text)
	if err != nil {
		return nil
	}
	return &date
}

type Dividend struct {
	Name             string     `json:"name"`
	Ticker           string     `json:"ticker"`
	ExDate           *time.Time `json:"exDate"`
	DividendRate     float32    `json:"dividendRate"`
	RecordDate       *time.Time `json:"recordDate"`
	PaymentDate      *time.Time `json:"paymentDate"`
	AnnouncementDate *time.Time `json:"announcementDate"`
}

func DividendsOnDate(date time.Time) ([]Dividend, error) {

	date = date.UTC().Truncate(24 * time.Hour)

	uri := fmt.Sprintf("https://api.nasdaq.com/api/calendar/dividends?date=%s", date.Format("2006-01-02"))

	// Craft the request for the page.
	req, _ := http.NewRequest("GET", uri, nil)
	req.Header.Set(
		"Accept",
		"application/json",
	)
	req.Header.Set(
		"User-Agent",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.0 Safari/605.1.15",
	)
	req.Header.Set("Accept-Language", "en-us")

	// Make the request.
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return []Dividend{}, err
	}
	defer resp.Body.Close()

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return []Dividend{}, err
	}

	// Define the structure of the response from the SeekingAlpha API endpoint.
	var body struct {
		Data struct {
			Calendar struct {
				Rows []struct {
					CompanyName      string
					Symbol           string
					DividendExDate   string  `json:"dividend_Ex_Date"`
					DividendRate     float32 `json:"dividend_Rate"`
					RecordDate       string  `json:"record_Date"`
					PaymentDate      string  `json:"payment_Date"`
					AnnouncementDate string  `json:"announcement_Date"`
				}
			}
		}
	}

	if err := json.Unmarshal(content, &body); err != nil {
		return []Dividend{}, err
	}

	dividends := []Dividend{}
	for _, item := range body.Data.Calendar.Rows {

		dividends = append(dividends, Dividend{
			Name:             item.CompanyName,
			Ticker:           item.Symbol,
			ExDate:           ParseDate(item.DividendExDate),
			DividendRate:     item.DividendRate,
			RecordDate:       ParseDate(item.RecordDate),
			PaymentDate:      ParseDate(item.PaymentDate),
			AnnouncementDate: ParseDate(item.AnnouncementDate),
		})
	}

	return dividends, err
}

var Cmd = &cobra.Command{
	Use:   "dividends",
	Short: "Scrape company dividend data",
	Run: func(cmd *cobra.Command, args []string) {

		today := time.Now().UTC().Truncate(24 * time.Hour)

		earnings, err := DividendsOnDate(today)
		if err != nil {
			log.Fatal(err)
		}

		for _, item := range earnings {
			// Create the event
			dividend := client.Dividend{
				Name:             item.Name,
				Ticker:           item.Ticker,
				ExDate:           item.ExDate,
				DividendRate:     item.DividendRate,
				RecordDate:       item.RecordDate,
				PaymentDate:      item.PaymentDate,
				AnnouncementDate: item.AnnouncementDate,
			}

			// Send it!!
			if _, err := client.Default().EmitDividendEvent("feeds.nasdaq.dividends", dividend); err != nil {
				log.Fatal(err)
			} else {
				log.Printf("%s paid dividends on %s", dividend.Ticker, dividend.ExDate)
			}
		}

		fmt.Println(len(earnings), "paid dividends on", today.Format("2006-01-02"))
	},
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.feeds.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// Cmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
