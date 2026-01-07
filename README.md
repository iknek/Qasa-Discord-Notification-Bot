# What is this?
Welp, I've been looking for an apartment, and whilst Qasa has a system for sending users email notifications about new listings for saved searches, I'd like to a. be notified asap, b. not flood my inbox with what I presume is copious amounts of advertisments. 

So, for lack of a Qasa mobile app, I decided to make this little Discord bot, which pings Qasa's backend GraphQL API every minute, and then sends a notification to a Discord server channel every time it sees a new apartment be posted. 

# How's it work?

Qasa doesn't have an API that's open to devs. But using super cool hacking skills (i.e., clicking F12 in my browser and going to the network tab), its clear that every time you do a search on the site, it fires off a querty (POST request) to a GraphQL backend server.

So, I did the lazy thing and just filtered the search on the Qasa website, then copied the GraphQL query it generated, and added it as a HTTP POST here in the code. This is sent every 60 seconds, giving me a JSON list of the properties for rent. 

Using some vibe-coding, I then simply extract the fields I want, making a listing object for each listing, and then format and send it to a server channel in Discord.

# How do I run this?

1. Setup a Discord bot, and invite it to your server. Don't forget to tick "bot" in the OAuth2 URL Generator, otherwise the bot won't be able to join your server!
2. Edit 'query' in getListings() to be whatever you want.
3. Do 'go build' to build an executable.
4. Run it using .\qasabot.exe -t "bot token" -c "channel to update adverts in" (enable dev options in Discord, then right click on a channel to get its ID)

# Am I allowed to do this?
I don't know, I've sent an email to Qasa re. this, and if they don't want me releasing it publicly, or in the future want it removed, they're welcome to reach out to me. 

As a common courtesy, if you are going to use this, please don't ping the backend every second, that's both pointless, and will flood their site for no reason... 

# Why doesn't this work for me?
I don't know, it works on my machine (for now). This will probably break if Qasa ever decides to change their backend, or adds some sort of bot detection/filter.