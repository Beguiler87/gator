# gator
RSS Aggregator Project (Go)
To use this system, you will need to have both Postgres and Go installed.
To install gator you'll need to run the following command from your terminal: go install https://github.com/Beguiler87/gator
Once installed, be sure to migrate up so that the database functions properly. Run the following command from your terminal: goose -dir sql/schema postgres "postgres://postgres:postgres@localhost:5432/gator" up
Once the migration is complete, you'll need to create a file in your home directory called .gatorconfig.json. Enter the text below in that file, replacing username with the user name you wish to identify yourself with. This is the touchstone your instance of gator will run off of.
{
  "db_url": "postgres://postgres:postgres@localhost:5432/gator",
  "current_user_name": "username"
}
Once the touchstone file is saved, you're ready to use gator. Prefix all commands with "gator ".
Example commands:
addfeed <url>: adds a new feed to your database.
agg <x>s: provides an updated list of articles from your followed feeds, refreshing every x seconds. Replace x with the interval you wish to use. This continues until stopped with "Ctrl + c"
browse: allows browsing of the currently followed feeds.
following: displays a list of all currently followed feeds.