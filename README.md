# Gator CLI
Gator is a command-line RSS feed aggregator written in Go and backed by a PostgreSQL database. It allows you to register users, follow RSS feeds, and browse the latest published posts directly from your terminal.

## Prerequisites:
Before installing and running Gator, ensure you have the following software installed on your system:
- Go: Version 1.22 or higher is recommended.
- PostgreSQL: A running Postgres instance to store users, feeds, and posts.

## Installation:
You can install the Gator CLI globally using the standard Go installation command:

```
go install github.com/slonepearson/gator/cmd/gator@latest
```

Note: Make sure your GOPATH/bin directory is added to your system's PATH environment variable to run the gator command from anywhere.

## Configuration Setup:
Gator relies on a JSON configuration file to know which database to connect to and which user is currently logged in.

1. Create a file named .gatorconfig.json in your user home directory (e.g., ~/.gatorconfig.json on macOS/Linux or C:\Users\<YourUsername>\.gatorconfig.json on Windows).
2. Populate the file with your PostgreSQL connection string and an initial empty string for the current user:

```json
{
  "db_url": "postgres://username:password@localhost:5432/gator_db?sslmode=disable",
  "current_user_name": "",
  "last_read_top": "",
  "last_read_top_uuid": "",
  "last_read_bottom": "",
  "last_read_bottom_uuid": "",
}
```

## Running the Program:
Once installed and configured, run the application by executing the `gator` command followed by a sub-command.

### Available Commands:
Here are some of the key commands you can run to interact with the application:

* Register a new user:
``` 
gator register <username>
```
* Log in as an existing user:
```
gator login <username>
```
* List all registed users:
```
gator users
```
* Add and follow a new RSS feed:
```
gator addfeed "Tech News" "https://example.com"
```
* List all saved feeds:
```
gator feeds
```
* Aggregate posts from feeds:
Has to be atlease 1 minute
```
gator agg <time between requests>
```
* Browse through saved posts:
```
gator browse 
```
To browse the next latest posts
```
gator browse --next
```
To browse the previous viewed posts
```
gator browse --prev
```
To set the number of posts you want returned
```
gator browse --limit <number>
```