# newsfeed [![build](https://github.com/Semior001/newsfeed/actions/workflows/.go.yaml/badge.svg)](https://github.com/Semior001/newsfeed/actions/workflows/.go.yaml) [![codecov](https://codecov.io/gh/Semior001/newsfeed/branch/master/graph/badge.svg?token=0MAV99RJ1C)](https://codecov.io/gh/Semior001/newsfeed)
a bot that sends news from the feed in a compressed format

## options
```
Application Options:
      --json-logs              turn on json logs [$JSON_LOGS]
      --dbg                    turn on debug mode [$DEBUG]

Help Options:
  -h, --help                   Show this help message

[run command options]
          --timeout=           timeout for http calls to articles (default: 5s) [$TIMEOUT]
          --admin-ids=         admin IDs [$ADMIN_IDS]
          --auth-token=        token for authorizing requests [$AUTH_TOKEN]
          --store-path=        parent dir for bolt files [$STORE_PATH]

    telegram:
          --telegram.token=    telegram token [$TELEGRAM_TOKEN]

    openai:
          --openai.token=      OpenAI token [$OPENAI_TOKEN]
          --openai.max-tokens= max tokens for OpenAI (default: 1000) [$OPENAI_MAX_TOKENS]
```
