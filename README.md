Experimental version of Drone.IO with deep GitHub, GitHub Enterprise and Bitbucket integration.

I am currently copy / pasting functionality into this branch. So if you see something that is missing it
probably because I haven't gotten to that section yet. The initial focus has been on the API and UI, which
means builds are not hooked up yet. Help is of course welcome.

Some of the fundamental changes include

1. modification to project structure
2. api-driven design
3. interface to abstract github, bitbucket, gitlab code (see /shared/remote)
4. handlers are structures with service providers "injected"
5. github, bitbucket, etc native permissions are used. No more teams or permissions in Drone
6. github, bitbucket, etc authentication is used. No more drone password
7. github, bitbucket, etc repository data is cached upon login (and subsequent logins)
8. configuration is driven by a file (~/.drone/drone.toml) and not the database

... probably more that I'm forgetting

Normally I wouldn't post experimental code in such disarray, but given the amount of activity around
the project I wanted to give the community visibility into these changes. I could also use the help!
