# newsfeed [![build](https://github.com/Semior001/newsfeed/actions/workflows/.go.yaml/badge.svg)](https://github.com/Semior001/newsfeed/actions/workflows/.go.yaml) [![codecov](https://codecov.io/gh/Semior001/newsfeed/branch/master/graph/badge.svg?token=0MAV99RJ1C)](https://codecov.io/gh/Semior001/newsfeed)
a bot that sends news from the feed in a compressed format

## options
```
Application Options:
      --json-logs                      turn on json logs [$JSON_LOGS]
      --dbg                            turn on debug mode [$DEBUG]

Help Options:
  -h, --help                           Show this help message

[run command options]
          --store-path=                parent dir for bolt files [$STORE_PATH]

    bot:
          --bot.timeout=               timeout for requests (default: 6m) [$BOT_TIMEOUT]
          --bot.admin-ids=             admin IDs [$BOT_ADMIN_IDS]
          --bot.auth-token=            token for authorizing requests [$BOT_AUTH_TOKEN]

    telegram:
          --bot.telegram.token=        telegram token [$BOT_TELEGRAM_TOKEN]

    openai:
          --revisor.openai.token=      OpenAI token [$REVISOR_OPENAI_TOKEN]
          --revisor.openai.max-tokens= max tokens for OpenAI (default: 1000) [$REVISOR_OPENAI_MAX_TOKENS]
          --revisor.openai.timeout=    timeout for OpenAI calls (default: 5m) [$REVISOR_OPENAI_TIMEOUT]
```
