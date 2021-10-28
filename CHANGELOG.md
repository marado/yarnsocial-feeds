
<a name="0.1.0"></a>
## 0.1.0 (2021-10-28)

### Bug Fixes

* Fix Dockerfile image and add /health endpoint
* Fix content processor and increase maxLength to 576
* Fix import paths
* Fix a possible crash when feed.Image is nil
* Fix invalid log.Error/log.Errorf calls
* Fix bug calculating min
* Fix cron expr for TikTok bot
* Fix TikTok cron expr
* Fix cron expr for TikTOk bot
* Fix bug in calcualting morning vs. afternoon vs. evening
* Fix cron expr for TikTok bot to run every hour on the half hour instead
* Fix we-are-feeds handler to pull feed names from data directory
* Fix time symbol map
* Fix @tiktok job interval
* Fix typo
* Fix ci
* Fix ci build
* Fix placeholder for URL input
* Fix feed name validation and restrict to 25 characters long to prevent abuse
* Fix a bug where items with no pubhlished date cause a panic
* Fix BaseURL calculation of feeds text/plain output
* Fix Makefile dev target
* Fix footer in templates
* Fix typos and expand landing page text
* Fix bug skipping invalid feeds when getting feed stats
* Fix Docker image to have valid CA Root certs
* Fix a nil map bug when adding feeds to a fresh config
* Fix missing default config for Docker image

### Features

* Add empty CHANGELOG
* Add GoReleaser config
* Add a shameless promotion of Yarn.social in all feeds preamble
* Add avatar to @tiktok bot
* Add support for avatar hashes to cache bust Yarn.social pods
* Add a working docker-compose
* Add support for aggregating Twitter timelines into Twtxt feeds (#12)
* Add Drone CI config
* Add daily RotateFeedsJob
* Add Content-Type headers to text handlers
* Add a simple @tiktok builtin bot
* Add HEAD handling
* Add support for feed avatars
* Add feed name and url validation
* Add we-are-feeds endpoint and content negogation foe /feeds (#8)
* Add CI/CD (#7)
* Add LICENSE (#6)
* Add README (#5)
* Add improved teemplates that improve the UI/UX
* Add build tools for building docker images and releasing github releases ready for production
* Add web application/service for public consumption

### Updates

* Update deps
* Update evening symbol
* Update README.md

