package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/chromedp/chromedp"
	"github.com/gocolly/colly/v2"
	"github.com/ozdemirrulass/mamarkt-scraper/db"
	"github.com/ozdemirrulass/mamarkt-scraper/util"
)

type Product struct {
	Name      string
	CreatedAt string
	Targets   []Target
}

type Target struct {
	Url       string
	PriceLogs []PriceLog
}

type PriceLog struct {
	Price     float64
	Currency  string
	ScannedAt string
}

func main() {
	db.Init()
	db := db.GetDB()

	collectedXMLs, err := collectXML("https://www.migros.com.tr/hermes/api/sitemaps/sitemap.xml", "//sitemap/loc")
	if err != nil {
		log.Fatalf("Error collecting XMLs: %v", err)
	}

	filteredXMLs := util.FilterXMLs(collectedXMLs, "product")

	var collectedProductUrls []string
	for _, v := range filteredXMLs {
		collectedProductUrlBatch, err := collectXML(v, "//url/loc")
		if err != nil {
			log.Fatalf("Error collecting product URLs: %v", err)
		}
		collectedProductUrls = append(collectedProductUrls, collectedProductUrlBatch...)
	}

	limiter := make(chan int, 25)
	products := make(chan Product)
	BatchSize := 25
	batchChannel := make(chan []Product, BatchSize)
	var batch []Product

	go func() {
		for product := range products {
			batch = append(batch, product)
			if len(batch) == BatchSize {
				batchChannel <- batch
				batch = []Product{}
			}
		}

		if len(batch) > 0 {
			batchChannel <- batch
		}
		close(batchChannel)
	}()

	for i := 0; i < 10; i++ {
		go func(db *dynamodb.DynamoDB) {
			for batch := range batchChannel {
				var items []map[string]*dynamodb.AttributeValue
				for _, product := range batch {
					currentTime := time.Now()
					product.CreatedAt = currentTime.Format("2006-01-02 15:04:05.000")

					item, err := dynamodbattribute.MarshalMap(product)
					if err != nil {
						log.Fatalln(err)
					}
					items = append(items, item)
				}
				writeRequests := make([]*dynamodb.WriteRequest, len(items))
				for i, item := range items {
					writeRequests[i] = &dynamodb.WriteRequest{
						PutRequest: &dynamodb.PutRequest{
							Item: item,
						},
					}
				}

				input := &dynamodb.BatchWriteItemInput{
					RequestItems: map[string][]*dynamodb.WriteRequest{
						"products": writeRequests,
					},
				}
				_, err := db.BatchWriteItem(input)
				if err != nil {
					log.Println(err)
				} else {
					log.Println("New batch created in the database")
				}
			}
		}(db)
	}

	for _, v := range collectedProductUrls {
		limiter <- 1
		go func(v string) {
			product, err := scrapeProduct(v)
			if err != nil {
				log.Printf("Error scraping product: %v", err)
			}
			products <- product
			defer func() {
				<-limiter
			}()
		}(v)
	}
}

func collectXML(url string, pattern string) ([]string, error) {
	var collectedXMLs []string

	c := colly.NewCollector()

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL)
	})

	c.OnError(func(_ *colly.Response, err error) {
		log.Println("Error collecting XML:", err)
	})

	c.OnResponse(func(r *colly.Response) {
		fmt.Println("Visited", r.Request.URL)
	})

	c.OnXML(pattern, func(e *colly.XMLElement) {
		collectedXMLs = append(collectedXMLs, e.Text)
	})

	c.OnScraped(func(r *colly.Response) {
		fmt.Println("Finished", r.Request.URL)
	})

	if err := c.Visit(url); err != nil {
		return nil, err
	}

	return collectedXMLs, nil
}

func scrapeProduct(url string) (Product, error) {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	var priceAndCurrency string
	var product Product

	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.Text(".amount", &priceAndCurrency, chromedp.ByQuery),
		chromedp.Text(".product-details h3", &product.Name, chromedp.ByQuery),
	)
	if err != nil {
		return product, err
	}

	currentTime := time.Now()
	scannedAt := currentTime.Format("2006-01-02 15:04:05.000")

	price, currency, _ := util.SplitPriceAndCurrency(priceAndCurrency)
	priceLog := PriceLog{
		Price:     price,
		Currency:  currency,
		ScannedAt: scannedAt,
	}
	target := Target{
		Url:       url,
		PriceLogs: []PriceLog{priceLog},
	}
	product.Targets = append(product.Targets, target)

	return product, nil
}
