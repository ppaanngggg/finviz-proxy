# finviz-proxy

Unoffical APT for Finviz screener application. Finviz has a great screener application. 

## Usage

1. simple run the `main.go` file, it will default listen on port 8000.
2. use docker to run the image `ppaanngggg/finviz-proxy`.
3. use my RapidAPI service, [finviz-screener](https://rapidapi.com/ppaanngggg/api/finviz-screener).

## Envs

1. `LOG_COLOR` default:"true", enable color log.
2. `PORT` default:"8000", listen port.
3. `TIMEOUT` default:"60s", http client timeout.
4. `THROTTLE` default:"100", max concurrent request.

## API

### Get Parameters

**Request:**

```http request
GET /params
```

**Response:**

1. Sorters, decide how to sort result
2. Signals, finviz defined signal filter
3. Filters, all available filters of finviz screener

```json
{
	"filters": [
		{
			"name": "Exchange",
			"description": "Stock Exchange at which a stock is listed.",
			"options": [
				{
					"name": "AMEX",
					"value": "exch_amex"
				},
                ...
			]
		},
		{
			"name": "Index",
			"description": "A major index membership of a stock.",
			"options": [
				{
					"name": "S&P 500",
					"value": "idx_sp500"
				},
				{
					"name": "NASDAQ 100",
					"value": "idx_ndx"
				},
				...
			]
		},
		...
	],
	"sorters": [
		{
			"name": "Ticker",
			"value": "ticker"
		},
		{
			"name": "Tickers Input Filter",
			"value": "tickersfilter"
		},
		...
	],
	"signals": [
		{
			"name": "Top Gainers",
			"value": "ta_topgainers"
		},
		{
			"name": "Top Losers",
			"value": "ta_toplosers"
		},
		...
	]
}
```

### Get Table

**Request:**

Fetch the table of screener. You can use any values from API `/params` to control your screener.

```http request
GET /table?order=company&desc=true&signal=ta_mostactive&filters[0]=exch_nasd&filters[1]=idx_sp500
```

**Response:**

1. `headers`, list of string, fetch from webpage's table;
2. `rows`, list of tuple, each line is a ordered record from webpage's table;

```json
{
	"headers": [
		"No.",
		"Ticker",
		"Company",
		"Sector",
		"Industry",
		"Country",
		"Market Cap",
		"P/E",
		"Price",
		"Change",
		"Volume"
	],
	"rows": [
		[
			"1",
			"AMD",
			"Advanced Micro Devices, Inc.",
			"Technology",
			"Semiconductors",
			"USA",
			"171.55B",
			"-",
			"107.21",
			"-2.03%",
			"12,136,988"
		],
		[
			"2",
			"GOOG",
			"Alphabet Inc.",
			"Communication Services",
			"Internet Content & Information",
			"USA",
			"1677.43B",
			"30.39",
			"133.40",
			"0.14%",
			"2,499,670"
		],
		...
	]
}
```
