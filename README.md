# Price Proxy

`priceproxy` fetches prices periodically from sources as listed in its config file. It provides a simple REST API for fetching prices.

Vega trading bots use the price proxy to place orders with real-world prices while avoiding hitting API rate limits on upstream price sources. The bots can then provide liquidity on Testnet, and help make the charts look realistic.

## Install - from a release

See the [releases page](https://github.com/vegaprotocol/priceproxy/releases/).

## Install - from source

Either run `go get`:

```bash
go get github.com/vegaprotocol/priceproxy/cmd/priceproxy@latest
```

Or clone the repository:

```bash
git clone https://github.com/vegaprotocol/priceproxy.git
cd priceproxy
go install ./cmd/priceproxy
```

A compiled `priceproxy` binary should now be in `$GOPATH/bin`.

See also:
- `Makefile` - contains useful commands for running builds, tests, etc.

## Running

```bash
priceproxy -config /path/to/your/config.yml
```

## Config

Save the following as `config.yml`:

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

## Supported price sources

The following price sources are currently supported. Pull requests are gratefully received for more sources.

- [Bitstamp](https://www.bitstamp.net/), [API docs](https://www.bitstamp.net/api/), see `pricing/bitstamp.go`
- [Coin Market Cap](https://coinmarketcap.com/), [API docs](https://coinmarketcap.com/api/documentation/v1/), see `pricing/coinmarketcap.go`
- [FTX](https://ftx.com/), [REST API docs](https://docs.ftx.com/#rest-api), see `pricing/ftx.go`

## priceproxy API Endpoints

| Method     | Location                               | Description                               |
| :--------- | :------------------------------------- | :---------------------------------------- |
| GET        | `/prices?params...`                    | List some/all prices                      |
| GET        | `/sources`                             | List all sources                          |
| GET        | `/sources/`[**name** _string_]         | List one source                           |
| GET        | `/status`                              | Resturn status=true                       |

### Query parameters for `GET /prices`

- **source** _string_: Limit the results to ones with the given source.
- **base** _string_: Limit the results to ones with the given base.
- **quote** _string_: Limit the results to ones with the given quote.
- **wander** _bool_: Limit the results to ones with the given wander setting.

## Licence

Distributed under the MIT License. See `LICENSE` for more information.
