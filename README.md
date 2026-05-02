# quantToGo

> Portfolio-facing version of a trading-system project, with only publicly shareable details retained.

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org)

## Context

This repository is a portfolio adaptation of prior real-world work, not a full commercial codebase.
Because the original project involved company IP and compliance constraints, this version is intentionally scoped:

- Sensitive business details are removed.
- Strategy parameters, risk thresholds, and deployment specifics are not the original production values.
- Documentation focuses on architecture and implementation approach, not internal operational details.
- This project is finished with claude, gpt, gemini and other llm models. All parts are manually checked but a double check is still needed if you are gonna use it.


## Current Status (Transparent Version)

This version is suitable for demonstration and technical discussion, but it is **not** a plug-and-play production system.

### Implemented

- Core module structure (data, strategy, execution, risk)
- Partial Binance Futures integration
- Basic startup and status-check scripts

### Known Issues

- Error handling for some edge cases is incomplete (especially after network jitter/reconnect state transitions).
- Parts of the documentation and code are not perfectly synchronized.
- Backtesting, monitoring, and alerting are currently incomplete.
- The method to place orders doesn't works properly. Some orders like market and limit use order and others use algoOrder.

### Not Yet Implemented / Planned

- More robust backtesting and parameter evaluation tooling
- Finer-grained risk-control mechanisms
- Unified monitoring dashboard and alert channels (e.g., Telegram/email)
- Multi-strategy parallel execution with isolation

## Quick Start (For Local Understanding)

```bash
# 1) Check configuration
vim config/config.yaml

# 2) Start
./scripts/start-live.sh

# 3) Check status
./scripts/check-status.sh
```

> Use a test environment first. Do not start with live capital.

## Documentation

- [Docs Overview](docs/README.md)
- [Quick Start](docs/01-QUICK_START.md)
- [Architecture](docs/02-ARCHITECTURE.md)
- [Strategy](docs/03-STRATEGY.md)
- [API Reference](docs/API_REFERENCE.md)
- [Changelog](docs/CHANGELOG.md)

## Risk & Disclaimer

- This repository is for technical demonstration and learning only, not investment advice.
- Live trading carries material risk. Test thoroughly and assess risk before any real deployment.
- This is a sanitized portfolio version by design; omitted details are intentional.

## License

MIT
