
# Gator

Gator is a command-line RSS feed aggregator written in Go. It allows users to register accounts, add RSS feeds, follow feeds, aggregate posts into a PostgreSQL database, and browse the latest posts.

## Prerequisites

Before using Gator, make sure you have the following installed:

Go (version 1.24 or newer recommended)
PostgreSQL

You'll also need a running PostgreSQL server and a database for Gator.

## Installation

Clone the repository:

git clone https://github.com/CKojiro/go_aggregator.git
cd go_aggregator

Install the CLI:

go install

This will build and install the gator executable.

## Database Setup

Create a PostgreSQL database (for example, gator).

Run the Goose migrations:

goose -dir sql/schema postgres "postgres://postgres:postgres@localhost:5432/gator" up

Replace the connection string with your own if necessary.

## Configuration

Create a file named:

~/.gatorconfig.json

Example contents:

{
  "db_url": "postgres://postgres:postgres@localhost:5432/gator?sslmode=disable",
  "current_user_name": ""
}

Replace the database URL with your own PostgreSQL connection string.

## Usage

Register a new user:

gator register alice

Log in:

gator login alice

Add a feed:

gator addfeed "Boot.dev Blog" https://blog.boot.dev/index.xml

Follow a feed:

gator follow https://blog.boot.dev/index.xml

Show the feeds you're following:

gator following

Aggregate feeds:

gator agg 30s

Browse the latest posts:

gator browse

Browse more posts:

gator browse 10

Unfollow a feed:

gator unfollow https://blog.boot.dev/index.xml

Reset the database:

gator reset
Available Commands
register <username> — Create a new user.
login <username> — Log in as an existing user.
users — List all registered users.
addfeed <name> <url> — Add a new RSS feed.
feeds — List all feeds.
follow <url> — Follow a feed.
following — List feeds followed by the current user.
unfollow <url> — Stop following a feed.
agg <duration> — Continuously fetch new posts from feeds.
browse [limit] — Display the newest posts.
reset — Delete all users and associated data.
Notes
The application stores its configuration in ~/.gatorconfig.json.
Feed aggregation stores new posts in PostgreSQL.
Running agg will continue fetching feeds until the program is stopped (Ctrl+C).
