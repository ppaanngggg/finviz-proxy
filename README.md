# **Finviz Proxy**

🐳 **[Docker Hub](https://hub.docker.com/r/ppaanngggg/finviz-proxy) | 🐙 [RapidAPI](https://rapidapi.com/ppaanngggg/api/finviz-screener)**

- 👏 **Update 7/23/2024 - We refactor new /table_v2 API and offer python script turn params into openapi schema.**
- 👏 **Update 6/11/2024 - We now support login with your own [Elite](https://finviz.com/elite.ashx?a=611157936) account and can fetch Elite's real-time data.**

[Finviz](https://finviz.com/?a=611157936) offers a fantastic screener application, but it lacks exposed APIs and is server-rendered. Therefore, I developed a server to fetch pages from Finviz and parse them to extract relevant information. I hope this can assist you in your financial research.

## **Usage**

1. Simply run the `main.go` file. By default, it will listen on port 8000.
2. Use Docker to run the image `docker run -p 8000:8000 ppaanngggg/finviz-proxy`.
3. Utilize my RapidAPI service, [Finviz Screener](https://rapidapi.com/ppaanngggg/api/finviz-screener).

## **Environments**

### Serve Relative

1. `PORT` (default: 8000) - the listening port.
2. `TIMEOUT` (default: 60s) - this is the http client timeout.
3. `THROTTLE` (default: 100) - this represents the maximum number of concurrent requests.
4. `CACHETTL` (default: 60s) - this is the table cache timeout.

### Elite Relative

1. `ELITELOGIN` (default: false) - determines if Elite Account login is enabled.
2. `EMAIL` (default: ) - email of your Elite Account.
3. `PASSWORD` (default: ) - password of your Elite Account.

## **API**

### **1. Get Parameters**

This endpoint provides all the necessary parameters to make requests to the Finviz screener.

**Request:**

No parameters required.

```bash
# curl example
curl localhost:8000/params
```

**Response:**

1. `sorters` - determines the sorting method for results.
2. `signals` - a special filter defined by Finviz for signals.
3. `filters` - all available filters of the Finviz screener.

```json
// output sample of `/params`
{
	"filters": [
		{
			"id": "fs_exch",
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
			"id": "fs_idx",
			"name": "Index",
			"description": "A major index membership of a stock.",
			"options": [
				{
					"name": "S&P 500",
					"value": "idx_sp500"
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
		...
	],
	"signals": [
		{
			"name": "Top Gainers",
			"value": "ta_topgainers"
		},
		...
	]
}
```

### **2. Get Table**

This endpoint returns a structured table with the screener results based on the provided parameters.

You can use any `value` from the API response of `/params` to manage your `/table` response.

**Request:**

**🥳 New V2**

Send a `POST` request to `/table_v2` , the field of request body are:

1. `order`: Select values from `sorters`. For example: `"order": ticker`.
2. `desc`: Set to `true` or `false` to control the sort order. For example, `"desc": true`.
3. `signal`: Select values from `signals`. For example, `"signal": ta_topgainers`.
4. `filters` : Select id and value from `filters`. For example, `"fs_exch": "exch_nasd"`.

```bash
curl -XPOST 'http://localhost:8000/table_v2' --data '{
  "order": "ticker",
  "desc": true,
  "signal": "ta_topgainers",
  "filters": {
    "fs_exch": "exch_nasd"
  }
}'
```

**⛔ Deprecated V1**

Send a `GET` request to `/table`. The supported parameters are:

1. `order`: Select values from `sorters`. For example: `order=ticker`.
2. `desc`: Set to `true` or `false` to control the sort order. For example, `desc=true`.
3. `signal`: Select values from `signals`. For example, `signal=ta_topgainers`.
4. `filters`: Filters offer various options and can accept multiple values. Select values from `filters`. For instance, use `filters=exch_nasd` for a single value or `filters=exch_nasd&filters=idx_sp500` for multiple filters.

```bash
curl 'localhost:8000/table?order=ticker&desc=true&signal=ta_topgainers&filters=exch_nasd&filters=idx_sp500'
```

**Response:**

1. `headers`: A list of strings representing the headers fetched from a webpage's table.
2. `rows`: A list of tuples, where each tuple is an ordered record fetched from a webpage's table.

```json
// output example
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
      "FSLR",
      "First Solar Inc",
      "Technology",
      "Solar",
      "USA",
      "31.53B",
      "30.88",
      "294.53",
      "5.26%",
      "4,099,119"
    ],
    [
      "2",
      "AAPL",
      "Apple Inc",
      "Technology",
      "Consumer Electronics",
      "USA",
      "3176.46B",
      "32.21",
      "207.15",
      "7.26%",
      "172,010,601"
    ]
  ]
}
```