gofer {

  origin "binance" {
    type = "tick_generic_jq"
    url  = "https://api.binance.com/api/v3/ticker/24hr"
    jq   = ".[] | select(.symbol == ($ucbase + $ucquote)) | {price: .lastPrice, volume: .volume, time: (.closeTime / 1000)}"
  }

  origin "binance_simple" {
    type = "tick_generic_jq"
    url  = "https://api.binance.com/api/v3/ticker/24hr?symbol=$${ucbase}$${ucquote}"
    jq   = "select(.symbol == ($ucbase + $ucquote)) | {price: .lastPrice, volume: .volume, time: (.closeTime / 1000)}"
  }

  origin "bitfinex" {
    type = "tick_generic_jq"
    url  = "https://api-pub.bitfinex.com/v2/tickers?symbols=ALL"
    jq   = ".[] | select(.[0] == \"t\" + ($ucbase + $ucquote) or .[0] == \"t\" + ($ucbase + \":\" + $ucquote) ) | {price: .[7], time: now|round, volume: .[8]}"
  }

  origin "bitfinex_simple" {
    type = "tick_generic_jq"
    url  = "https://api-pub.bitfinex.com/v2/tickers?symbols=t$${ucbase}$${ucquote}"
    jq   = "{price: .[][7], time: now|round, volume: .[][8]}"
  }

  origin "bitstamp" {
    type = "tick_generic_jq"
    url  = "https://www.bitstamp.net/api/v2/ticker/$${lcbase}$${lcquote}"
    jq   = "{price: .last, time: .timestamp, volume: .volume}"
  }

  origin "coinbase" {
    type = "tick_generic_jq"
    url  = "https://api.pro.coinbase.com/products/$${ucbase}-$${ucquote}/ticker"
    jq   = "{price: .price, time: .time, volume: .volume}"
  }

  origin "gemini" {
    type = "tick_generic_jq"
    url  = "https://api.gemini.com/v1/pubticker/$${lcbase}$${lcquote}"
    jq   = "{price: .last, time: (.volume.timestamp/1000), volume: .volume[$ucquote]|tonumber}"
  }

  origin "hitbtc" {
    type = "tick_generic_jq"
    url  = "https://api.hitbtc.com/api/2/public/ticker?symbols=$${ucbase}$${ucquote}"
    jq   = "{price: .[0].last|tonumber, time: .[0].timestamp|strptime(\"%Y-%m-%dT%H:%M:%S.%fZ\")|mktime, volume: .[0].volumeQuote|tonumber}"
    // An alternative approach without dealing with decimal seconds:
    //
    // jq = "{price: .[0].last|tonumber, time: .[0].timestamp|split(\".\")[0]|strptime(\"%Y-%m-%dT%H:%M:%S\")|mktime, volume: .[0].volumeQuote|tonumber}"
  }

  origin "huobi" {
    type = "tick_generic_jq"
    url  = "https://api.huobi.pro/market/tickers"
    jq   = ".data[] | select(.symbol == ($lcbase+$lcquote)) | {price: .close, volume: .vol, time: now|round}"
  }

  origin "ishares" {
    type = "ishares"
    url  = "https://ishares.com/uk/individual/en/products/287340/ishares-treasury-bond-1-3yr-ucits-etf?switchLocale=y&siteEntryPassthrough=true"
  }

  origin "kraken" {
    type = "tick_generic_jq"
    url  = "https://api.kraken.com/0/public/Ticker?pair=$${ucbase}/$${ucquote}"
    jq   = "($ucbase + \"/\" + $ucquote) as $pair | {price: .result[$pair].c[0]|tonumber, time: now|round, volume: .result[$pair].v[0]|tonumber}"
  }

  origin "kucoin" {
    type = "tick_generic_jq"
    url  = "https://api.kucoin.com/api/v1/market/orderbook/level1?symbol=$${ucbase}-$${ucquote}"
    jq   = "{price: .data.price, time: (.data.time/1000)|round, volume: null}"
  }

  origin "okx" {
    type = "tick_generic_jq"
    url  = "https://www.okx.com/api/v5/market/ticker?instId=$${ucbase}-$${ucquote}&instType=SPOT"
    jq   = "{price: .data[0].last|tonumber, time: (.data[0].ts|tonumber/1000), volume: .data[0].vol24h|tonumber}"
  }

  origin "upbit" {
    type = "tick_generic_jq"
    url  = "https://api.upbit.com/v1/ticker?markets=$${ucquote}-$${ucbase}"
    jq   = "{price: .[0].trade_price, time: (.[0].timestamp/1000), volume: .[0].acc_trade_volume_24h}"
  }

  data_model "AAVE/USD" {
    median {
      min_values = 3
      origin "coinbase" { query = "AAVE/USD" }
      origin "kraken" { query = "AAVE/USD" }
      origin "bitstamp" { query = "AAVE/USD" }
    }
  }

  data_model "AVAX/USD" {
    median {
      min_values = 3
      origin "coinbase" { query = "AVAX/USD" }
      origin "kraken" { query = "AVAX/USD" }
      origin "bitstamp" { query = "AVAX/USD" }
    }
  }

  data_model "BTC/USD" {
    median {
      min_values = 3
      origin "bitstamp" { query = "BTC/USD" }
      origin "coinbase" { query = "BTC/USD" }
      origin "gemini" { query = "BTC/USD" }
      origin "kraken" { query = "BTC/USD" }
    }
  }

  data_model "DAI/USD" {
    median {
      min_values = 3
      origin "kraken" { query = "DAI/USD" }
      origin "coinbase" { query = "DAI/USD" }
      origin "gemini" { query = "DAI/USD" }
    }
  }

  data_model "ETH/BTC" {
    median {
      min_values = 3
      origin "binance" { query = "ETH/BTC" }
      origin "bitstamp" { query = "ETH/BTC" }
      origin "coinbase" { query = "ETH/BTC" }
      origin "gemini" { query = "ETH/BTC" }
      origin "kraken" { query = "ETH/BTC" }
    }
  }

  data_model "ETH/USD" {
    median {
      min_values = 3
      origin "bitstamp" { query = "ETH/USD" }
      origin "coinbase" { query = "ETH/USD" }
      origin "gemini" { query = "ETH/USD" }
      origin "kraken" { query = "ETH/USD" }
    }
  }

  data_model "LINK/USD" {
    median {
      min_values = 3
      origin "coinbase" { query = "LINK/USD" }
      origin "kraken" { query = "LINK/USD" }
      origin "gemini" { query = "LINK/USD" }
      origin "bitstamp" { query = "LINK/USD" }
    }
  }

  data_model "MKR/USD" {
    median {
      min_values = 3
      origin "bitstamp" { query = "MKR/USD" }
      origin "coinbase" { query = "MKR/USD" }
      origin "gemini" { query = "MKR/USD" }
      origin "kraken" { query = "MKR/USD" }
    }
  }

  data_model "SOL/USD" {
    median {
      min_values = 3
      origin "coinbase" { query = "SOL/USD" }
      origin "kraken" { query = "SOL/USD" }
      origin "gemini" { query = "SOL/USD" }
    }
  }

  data_model "UNI/USD" {
    median {
      min_values = 3
      origin "coinbase" { query = "UNI/USD" }
      origin "kraken" { query = "UNI/USD" }
      origin "bitstamp" { query = "UNI/USD" }
    }
  }

  data_model "USDC/USD" {
    median {
      min_values = 3
      origin "kraken" { query = "USDC/USD" }
      origin "bitstamp" { query = "USDC/USD" }
      origin "gemini" { query = "USDC/USD" }
    }
  }

  data_model "USDT/USD" {
    median {
      min_values = 3
      origin "bitstamp" { query = "USDT/USD" }
      origin "bitfinex_simple" { query = "USDT/USD" }
      origin "coinbase" { query = "USDT/USD" }
      origin "kraken" { query = "USDT/USD" }
      origin "kucoin" { query = "USDT/USD" }
      origin "okx" { query = "USDT/USD" }
      origin "upbit" { query = "USDT/USD" }
    }
  }

  data_model "ADA/BTC" {
    median {
      min_values = 3
      origin "bitstamp" { query = "ADA/BTC" }
      origin "bitfinex_simple" { query = "ADA/BTC" }
      origin "kraken" { query = "ADA/BTC" }
      origin "okx" { query = "ADA/BTC" }
      origin "upbit" { query = "ADA/BTC" }
      origin "binance" { query = "ADA/BTC" }
      origin "hitbtc" { query = "ADA/BTC" }
      origin "huobi" { query = "ADA/BTC" }
    }
  }

  data_model "ADA/EUR" {
    median {
      min_values = 3
      origin "bitstamp" { query = "ADA/EUR" }
      origin "coinbase" { query = "ADA/EUR" }
      origin "kraken" { query = "ADA/EUR" }
      origin "binance_simple" { query = "ADA/EUR" }
    }
  }

  data_model "ADA/USD" {
    median {
      min_values = 3
      origin "bitstamp" { query = "ADA/USD" }
      origin "coinbase" { query = "ADA/USD" }
      origin "kraken" { query = "ADA/USD" }
      origin "bitfinex_simple" { query = "ADA/USD" }
      origin "hitbtc" { query = "ADA/USD" }
    }
  }

  // TUSD doesn't have enough sources. It can currently only be
  // retrieved from Kraken. The origins below are a useful reference
  // point for new data models as it tries to retrieve from all.
  data_model "TUSD/USD" {
    median {
      min_values = 1
      origin "binance" { query = "TUSD/USD" }
      origin "bitstamp" { query = "TUSD/USD" }
      origin "bitfinex_simple" { query = "TUSD/USD" }
      origin "coinbase" { query = "TUSD/USD" }
      origin "gemini" { query = "TUSD/USD" }
      origin "hitbtc" { query = "TUSD/USD" }
      origin "huobi" { query = "TUSD/USD" }
      origin "ishares" { query = "TUSD/USD" }
      origin "kraken" { query = "TUSD/USD" }
      origin "kucoin" { query = "TUSD/USD" }
      origin "okx" { query = "TUSD/USD" }
      origin "upbit" { query = "TUSD/USD" }
    }
  }

  // WMT is in the same boat.
  data_model "WMT/USD" {
    median {
      min_values = 1
      origin "binance" { query = "WMT/USD" }
      origin "bitstamp" { query = "WMT/USD" }
      origin "bitfinex_simple" { query = "WMT/USD" }
      origin "coinbase" { query = "WMT/USD" }
      origin "gemini" { query = "WMT/USD" }
      origin "hitbtc" { query = "WMT/USD" }
      origin "huobi" { query = "WMT/USD" }
      origin "ishares" { query = "WMT/USD" }
      origin "kraken" { query = "WMT/USD" }
      origin "kucoin" { query = "WMT/USD" }
      origin "okx" { query = "WMT/USD" }
      origin "upbit" { query = "WMT/USD" }
    }
  }
}

