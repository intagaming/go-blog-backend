# go-blog-backend

`go-blog-backend` is a blog API written in Go.

This repo is intended to be a learning project. My [blog][1] runs on MDX files,
so I don't need a backend.

Here's what I've done to the project so far:

1. I designed my API with OpenAPI.
1. I use [PlanetScale][2] as the MySQL Database as a Service.
1. I use [**no framework**][6], [**no ORM**][7].
1. I use Auth0 to authorize the API's users:
  - I utilize Auth0's Actions to customize the Access Token: In the Login flow,
    I added the email and name fields to register the user as an author of the
    API.
  - There are 2 roles: `author` and `admin`. `author` can only edit their posts.
    `admin` can edit all things.
1. I use [Zap][3] as the structured logger.
1. I use [chi][4] as the router.
1. I implemented a rate-limiting system with Redis, introduced by [this blog
   post][5].
1. I use Heroku to deploy, with Heroku Redis.

[1]: https://github.com/intagaming/blog2
[2]: https://planetscale.com
[3]: https://github.com/uber-go/zap
[4]: https://github.com/go-chi/chi
[5]: https://mauricio.github.io/2021/12/30/rate-limiting-in-go.html
[6]: https://www.stephengream.com/go-nethttp-vs-gin
[7]: https://eli.thegreenplace.net/2019/to-orm-or-not-to-orm