# 🖕 Zero Fucks Given as a Service

<p align="center">
  <!-- Placeholder for a logo if one existed, similar to the bunny -->
  <b>Zero Fucks Given as a Service (ZFGaas)</b>
</p>

Ever needed a graceful way to say "I absolutely do not care"?  
This tiny API returns random, unnecessarily verbose, and aggressively indifferent apologies — perfectly suited for any scenario where you need to formally apologize for not giving a single flying fuck.

Built for apathy, burnout, and humor.

> **Note:** This project is inspired by the legendary [no-as-a-service](https://github.com/hotheadhacker/no-as-a-service).

---

## 🚀 API Usage

**Base URL**

```
https://zfgaas.downormal.dev/sorry
```

**Method:** `GET`  
**Rate Limit:** `1 request per second` (Burst: 3)

### 🔄 Example Request

```http
GET /sorry
```

### ✅ Example Response

```json
{
  "reason": "I am profoundly sorry that your expectations and my level of fuck-giving never even briefly coexisted in this universe."
}
```

Use it in slack bots, auto-replies, or wherever you need to professionally dissociate.

---

## 🛠️ Self-Hosting

Want to run it yourself? It’s lightweight and simple.

### 1. Clone this repository

```bash
git clone https://github.com/anugrahsputra/zero-fucks-given-as-a-service.git
cd zero-fucks-given-as-a-service
```

### 2. Install dependencies

```bash
go mod tidy
```

### 3. Start the server

```bash
go run main.go
```

The API will be live at:

```
http://localhost:8080/sorry
```

---

## 📁 Project Structure

```
zero-fucks-given-as-a-service/
├── main.go             # Go Gin API
├── zero-fucks.json     # 500+ universal excuses/apologies
├── go.mod
├── go.sum
└── README.md
```

---

## 📦 go.mod

For reference, here’s the module config:

```go
module github.com/anugrahsputra/zero-fucks-given-as-a-service

go 1.25.5

require (
	github.com/gin-gonic/gin v1.11.0
	golang.org/x/time v0.14.0
)
```

---

## ⚠️ Disclaimer

This project is for satirical purposes. If you actually use this to reply to your boss, that's on you. I don't care.

---

## 📄 License

MIT — do whatever, just don’t say yes when you should say... well, nothing.
