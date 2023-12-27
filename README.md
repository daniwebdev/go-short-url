# GoShortURL- Open Source URL Shortener

GoShortURL is a powerful and flexible open-source URL shortener built with Go (Golang) and SQLite. It provides a simple and efficient way to create and manage short URLs, making it easy to share links and track their usage.

## Features

- **Custom Short URLs**: Create custom short URLs with ease.
- **Metadata Scraping**: Automatically fetches metadata (title, description, image) from the target URL.
- **Analytics**: Track the number of visits and last visit timestamp for each short URL.
- **Pagination**: Easily navigate through your list of short URLs with paginated results.
- **Secure**: Built with security in mind, ensuring reliable and safe URL shortening.

## Getting Started

### Prerequisites

Ensure you have the following prerequisites installed before setting up and running the URL shortener:

- Go (Golang)
- SQLite
- Additional Go libraries (dependencies):
  - [github.com/PuerkitoBio/goquery v1.8.1](https://pkg.go.dev/github.com/PuerkitoBio/goquery)
  - [github.com/andybalholm/cascadia v1.3.2](https://pkg.go.dev/github.com/andybalholm/cascadia)
  - [github.com/gorilla/mux v1.8.1](https://pkg.go.dev/github.com/gorilla/mux)
  - [github.com/mattn/go-sqlite3 v1.14.19](https://pkg.go.dev/github.com/mattn/go-sqlite3)
  - [golang.org/x/net v0.19.0](https://pkg.go.dev/golang.org/x/net)

You can install these dependencies using the `go get` command, for example:

```bash
go get -u github.com/PuerkitoBio/goquery@v1.8.1
go get -u github.com/andybalholm/cascadia@v1.3.2
go get -u github.com/gorilla/mux@v1.8.1
go get -u github.com/mattn/go-sqlite3@v1.14.19
go get -u golang.org/x/net@v0.19.0
```

This ensures all required libraries are installed for the URL shortener project.

Feel free to adjust the wording or formatting based on your preferences.

### Installation

Clone the repository:

```bash
git clone https://github.com/daniwebdev/go-short-url.git
cd go-short-url
```

Build and run the project:

```bash
GO_SHORT_KEY=your_api_key go run main.go
```

By default, the server will start on port `8080`. You can customize the port using the `-port` flag.

### Usage

#### Creating a Short URL

To create a short URL, send a POST request to the `/api` endpoint with a JSON payload:

```bash
curl -X POST http://localhost:8080/api -H "Content-Type: application/json" -H "X-API-Key: your_api_key" -d '{
  "url": "https://example.com",
  "custom_id": "custom_short_id",
}'
```

When creating a short URL, the metadata will be scraped from the provided URL, including title, description, and images, to enhance the information associated with the short URL. This ensures a richer preview when the short URL is accessed.

Feel free to customize the wording or provide additional details as needed.

#### Retrieving Short URLs

To get a list of short URLs with pagination, send a GET request to the `/api/{space}` endpoint:

```bash
curl http://localhost:8080/api/{space}?page=1&perPage=10 -H "Content-Type: application/json" -H "X-API-Key: your_api_key"
```

For more details on API endpoints, refer to the [API Rest](api.rest).

## Contributing

Contributions are welcome! Please check out our [Contribution Guidelines](CONTRIBUTING.md) for more details.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [Gorilla Mux](https://github.com/gorilla/mux) for the powerful router.
- [goquery](https://github.com/PuerkitoBio/goquery) - A great library for working with HTML documents using jQuery-style syntax.
- [cascadia](https://github.com/andybalholm/cascadia) - A CSS selector library for Go.
- [go-sqlite3](https://github.com/mattn/go-sqlite3) - SQLite driver for Go, ensuring seamless database operations.
- [golang.org/x/net](https://pkg.go.dev/golang.org/x/net) - The Go networking libraries providing support for various protocols.
