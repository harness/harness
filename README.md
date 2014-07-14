[![Build Status](http://test.drone.io/v1/badge/github.com/bradrydzewski/drone/status.svg?branch=exp)](http://test.drone.io/github.com/bradrydzewski/drone)

Experimental version of Drone.IO with deep GitHub, GitHub Enterprise and Bitbucket integration.

I am currently copy / pasting functionality into this branch. So if you see something that is missing it
probably because I haven't gotten to that section yet.

Some of the fundamental changes include

1. modification to project structure
2. api-driven design
3. interface to abstract github, bitbucket, gitlab code (see /shared/remote)
4. handlers are structures with service providers "injected"
5. github, bitbucket, etc native permissions are used. No more teams or permissions in Drone
6. github, bitbucket, etc authentication is used. No more drone password
7. github, bitbucket, etc repository data is cached upon login (and subsequent logins)
8. angularjs user interface with modified responsive design

... probably more that I'm forgetting

If you find an issue please don't log a bug. I'm probably aware of it and just haven't gotten to fixing it yet ... especially if it is related to a) angular b) emails or c) github, bitbucket and gitlab functionality.