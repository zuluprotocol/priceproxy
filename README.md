# Price Proxy

`priceproxy` fetches prices periodically from sources as listed in its config file. It provides a simple REST API for fetching prices.

Vega trading bots use the price proxy to place orders with real-world prices while avoiding hitting API rate limits on upstream price sources. The bots can then provide liquidity on Testnet, and help make the charts look realistic.

## Config

```yaml
server:
  listen: ":8080"
  logformat: text  # json, text
  loglevel: info  # debug, info, warn, error, fatal
  env: prod  # dev, prod

sources:

  - name: bitstamp
    sleepReal: 60  # seconds
    sleepWander: 2  # seconds
    url:
      scheme: https
      host: www.bitstamp.net
      path: "/api/v2/ticker/{base}{quote}/"
      # "{base}" and "{quote}" are replaced at runtime.

prices:

  - source: bitstamp
    base: BTC
    quote: USD
    factor: 1.0
    wander: true
```

## Endpoints

| Method     | Location                               | Description                               |
| :--------- | :------------------------------------- | :---------------------------------------- |
| GET        | `/prices?params...`                    | List some/all prices                      |
| GET        | `/sources`                             | List all sources                          |
| GET        | `/sources/`[**name** _string_]         | List one source                           |
| GET        | `/status`                              | Resturn status=true                       |

## Query parameters for `/prices`

- **source** _string_: Limit the results to ones with the given source.
- **base** _string_: Limit the results to ones with the given base.
- **quote** _string_: Limit the results to ones with the given quote.
- **wander** _bool_: Limit the results to ones with the given wander setting.
