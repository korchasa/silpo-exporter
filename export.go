package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"github.com/machinebox/graphql"
	"log"
	"os"
	"time"
)

var token string
var silpoTimeFormat = "2006-01-02T15:04:05.000Z07:00"

func main() {

	if "" == os.Getenv("ACCESS_TOKEN") {
		log.Fatal("ACCESS_TOKEN env var is empty")
	}
	token = os.Getenv("ACCESS_TOKEN")
	months := 24

	now := time.Now().UTC()
	currentYear, currentMonth, _ := now.Date()
	firstOfMonth := time.Date(currentYear, currentMonth, 1, 0, 0, 0, 0, now.Location())

	w := csv.NewWriter(os.Stdout)
	if err := w.Write([]string{"order", "time", "name", "count", "unit", "price"}); err != nil {
		log.Panic("error writing header to csv:", err)
	}
	for i := 0; i < months; i++ {
		from := firstOfMonth.AddDate(0, -i, 0)
		to := firstOfMonth.AddDate(0, -i + 1, 0)
		orders := getOrders(from, to)
		for _, order := range orders {
			items := getItems(&order)
			for _, item := range items {
				record := []string{
					order.ID,
					order.Created,
					item.Name,
					fmt.Sprintf("%f", item.Count),
					item.Unit,
					fmt.Sprintf("%f", item.Price),
				}
				if err := w.Write(record); err != nil {
					log.Fatalln("error writing item to csv:", err)
				}
			}
			w.Flush()
		}
	}
}

func getOrders(from time.Time, to time.Time) []Order {
	req := graphql.NewRequest(`
    query checks($offset: Int, $limit: Int, $dateFrom: DateTime, $dateTo: DateTime) {
		checks(offset: $offset, limit: $limit, dateFrom: $dateFrom, dateTo: $dateTo) {
		  id
		  created 
		  storeId 
		  __typename
		}
	}`)


	req.Var("offset", 1)
	req.Var("limit", 40)
	// Пришлось добавить миллисекунды, иначе сервер ругается
	req.Var("dateFrom", from.Format(silpoTimeFormat))
	req.Var("dateTo", to.Format(silpoTimeFormat))

	var respData OrdersResponse
	if err := request(req, &respData); err != nil {
		log.Panic(err)
	}
	return respData.Checks
}

func getItems(o *Order) []Item {
	req := graphql.NewRequest(`
    query check($storeId: ID!, $checkId: ID!, $creationDate: DateTime!) {
		check(storeId: $storeId, checkId: $checkId, creationDate: $creationDate) {
			items {
				name
				unit
				count
				price
				unitText
				__typename
		  	}
			__typename
		}
	}`)

	req.Var("storeId", o.StoreID)
	req.Var("checkId", o.ID)
	req.Var("creationDate", o.Created)

	var respData OrderItemsResponse
	if err := request(req, &respData); err != nil {
		log.Panic(err)
	}
	return respData.Check.Items
}

func request(req *graphql.Request, target interface{}) error {
	client := graphql.NewClient("https://silpo.ua/graphql")
	//client.Log = func(s string) { log.Println(s) }
	req.Header.Set("Access-Token", token)
	if err := client.Run(context.Background(), req, target); err != nil {
		return err
	}
	return nil
}

type OrdersResponse struct {
	Checks []Order `json:"checks"`
}

type Order struct {
	ID           string `json:"id"`
	Created      string `json:"created"`
	StoreID      int32 `json:"storeId"`
}

type OrderItemsResponse struct {
	Check struct {
		Items []Item `json:"items"`
	} `json:"check"`
}

type Item struct {
	Name     string `json:"name"`
	Unit     string `json:"unit"`
	Count    float32 `json:"count"`
	Price    float32 `json:"price"`
	//UnitText string `json:"unitText"`
}
