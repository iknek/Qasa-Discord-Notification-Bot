package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Variables used for command line parameters
var (
	Token     = ""
	ChannelID = ""                    // Channel ID where new apartment ads will be posted
	seenAds   = make(map[string]bool) // Track seen ad IDs
)

type Listing struct {
	ID           string
	Title        string
	Description  string
	Rent         int
	ImageURL     string
	Link         string
	Location     string
	RoomCount    float64
	StateDate    string
	SquareMeters int
}

func init() {
	flag.StringVar(&Token, "t", "", "Bot Token")
	flag.StringVar(&ChannelID, "c", "", "Channel ID for apartment notifications")
	flag.Parse()
}

func main() {
	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageCreate)

	// In this example, we only care about receiving message events.
	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentMessageContent

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	// Start the apartment monitoring goroutine
	go monitorApartments(dg)

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}

// messageCreate handles incoming messages
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Add any other message handling logic here if needed
}

// monitorApartments checks for new apartment ads every 60 seconds
func monitorApartments(s *discordgo.Session) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	// Initial population of seen ads and send them to channel
	listings, err := getListings()
	if err != nil {
		fmt.Println("Error getting initial listings:", err)
	} else {
		fmt.Printf("Initial scan found %d ads. Sending to channel...\n", len(listings))
		for _, listing := range listings {
			seenAds[listing.ID] = true
			// Send initial ads to channel
			if ChannelID != "" {
				sendNotification(s, listing, false)
				time.Sleep(1 * time.Second) // Small delay to avoid rate limiting
			} else {
				fmt.Println("Channel ID not set, skipping notification.")
			}
		}
		fmt.Printf("Initial scan complete. Tracking %d ads.\n", len(seenAds))
	}

	for range ticker.C {
		listings, err := getListings()
		if err != nil {
			fmt.Println("Error getting listings:", err)
			continue
		}

		for _, listing := range listings {
			if !seenAds[listing.ID] {
				// New ad found!
				seenAds[listing.ID] = true
				fmt.Printf("New ad found: (%s) %s\n", listing.ID, listing.Title)
				if ChannelID != "" {
					sendNotification(s, listing, true)
				}
			}
		}
	}
}

func formatStartDate(dateStr string) string {
	if dateStr == "" {
		return "Not specified"
	}

	// Parse the ISO 8601 date string
	t, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		return "Not specified"
	}

	// Get day with ordinal suffix (1st, 2nd, 3rd, etc.)
	day := t.Day()
	suffix := "th"
	switch day % 10 {
	case 1:
		if day != 11 {
			suffix = "st"
		}
	case 2:
		if day != 12 {
			suffix = "nd"
		}
	case 3:
		if day != 13 {
			suffix = "rd"
		}
	}

	return fmt.Sprintf("%d%s of %s", day, suffix, t.Format("January"))
}

// sendNotification sends an apartment listing to the Discord channel
func sendNotification(s *discordgo.Session, listing Listing, isNew bool) {
	fmt.Printf("Sending notification for listing ID %s\n", listing.ID)
	prefix := "🏠 **Apartment for rent**"
	if isNew {
		prefix = "🏠 **NEW Apartment for rent!**"
	}

	// Truncate description if it's too long
	description := listing.Description
	if len(description) > 500 {
		description = description[:500] + "..."
	}

	// Format the start date
	startDateFormatted := formatStartDate(listing.StateDate)

	embed := &discordgo.MessageEmbed{
		Title: listing.Title,
		URL:   listing.Link,
		Description: fmt.Sprintf("**Rent:** %d NOK/month\n**Location:** %s\n**Rooms:** %.0f\n**Size:** %d m²\n**Available from:** %s\n\n%s",
			listing.Rent, listing.Location, listing.RoomCount, listing.SquareMeters, startDateFormatted, description),
		Color: 0x00FF00, // Green color
		Image: &discordgo.MessageEmbedImage{
			URL: listing.ImageURL,
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	_, err := s.ChannelMessageSendComplex(ChannelID, &discordgo.MessageSend{
		Content: prefix,
		Embeds:  []*discordgo.MessageEmbed{embed},
	})
	if err != nil {
		fmt.Println("Error sending notification:", err)
	}
}

// getListings fetches apartment listings from the Qasa API
func getListings() ([]Listing, error) {
	baseURL := "https://api.qasa.se/graphql"

	// Build the request with GraphQL query parameters
	query := []byte(`{"operationName":"HomeSearch","variables":{"limit":60,"offset":0,"order":{"direction":"descending","orderBy":"published_or_bumped_at"},"params":{"homeType":["apartment","loft"],"shared":false,"maxMonthlyCost":20000,"currency":"NOK","areaIdentifier":["no/oslo"],"rentalType":["long_term"],"markets":["sweden","norway","finland"]}},"query":"query HomeSearch($order: HomeIndexSearchOrderInput, $offset: Int, $limit: Int, $params: HomeSearchParamsInput) {\n  homeIndexSearch(order: $order, params: $params) {\n    documents(offset: $offset, limit: $limit) {\n      hasNextPage\n      hasPreviousPage\n      nodes {\n        bedroomCount\n        blockListing\n        rentalLengthSeconds\n        householdSize\n        corporateHome\n        description\n        endDate\n        firstHand\n        furnished\n        homeType\n        id\n        instantSign\n        market\n        lastBumpedAt\n        monthlyCost\n        petsAllowed\n        platform\n        publishedAt\n        publishedOrBumpedAt\n        rent\n        currency\n        roomCount\n        seniorHome\n        shared\n        shortcutHome\n        smokingAllowed\n        sortingScore\n        squareMeters\n        startDate\n        studentHome\n        tenantBaseFee\n        title\n        wheelchairAccessible\n        finnishLandlordAssociation\n        location {\n          id\n          locality\n          countryCode\n          streetNumber\n          point {\n            lat\n            lon\n            __typename\n          }\n          route\n          __typename\n        }\n        displayStreetNumber\n        uploads {\n          id\n          order\n          type\n          url\n          __typename\n        }\n        __typename\n      }\n      pagesCount\n      totalCount\n      __typename\n    }\n    __typename\n  }\n}"}`)
	bodyReader := bytes.NewReader(query)
	req, err := http.NewRequest(http.MethodPost, baseURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Qasa-Discord-Bot/1.0")
	// Make the HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()
	// Parse the JSON response
	var result struct {
		Data struct {
			HomeIndexSearch struct {
				Documents struct {
					Nodes []struct {
						ID           string  `json:"id"`
						Title        string  `json:"title"`
						Description  string  `json:"description"`
						Rent         int     `json:"rent"`
						RoomCount    float64 `json:"roomCount"`
						SquareMeters int     `json:"squareMeters"`
						StartDate    string  `json:"startDate"`
						Location     struct {
							Locality string `json:"locality"`
							Route    string `json:"route"`
						} `json:"location"`
						Uploads []struct {
							URL   string `json:"url"`
							Order int    `json:"order"`
						} `json:"uploads"`
					} `json:"nodes"`
				} `json:"documents"`
			} `json:"homeIndexSearch"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	// Format the results
	var listings []Listing
	for _, node := range result.Data.HomeIndexSearch.Documents.Nodes {
		// Get the first image URL
		imageURL := ""
		if len(node.Uploads) > 0 {
			// Find the first image (order = 1 or lowest order)
			minOrder := node.Uploads[0].Order
			imageURL = node.Uploads[0].URL
			for _, upload := range node.Uploads {
				if upload.Order < minOrder {
					minOrder = upload.Order
					imageURL = upload.URL
				}
			}
		}

		// Build location string
		location := node.Location.Locality
		if node.Location.Route != "" {
			location = node.Location.Route + ", " + location
		}

		listing := Listing{
			ID:           node.ID,
			Title:        node.Title,
			Description:  node.Description,
			Rent:         node.Rent,
			ImageURL:     imageURL,
			Link:         fmt.Sprintf("https://qasa.se/home/%s", node.ID),
			Location:     location,
			RoomCount:    node.RoomCount,
			StateDate:    node.StartDate,
			SquareMeters: node.SquareMeters,
		}
		listings = append(listings, listing)
	}

	return listings, nil
}
