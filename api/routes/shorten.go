package routes

import (
	"os"
	"strconv"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
	"github.com/gondsuryaprakash/shortenurl/database"
	"github.com/gondsuryaprakash/shortenurl/helpers"
	"github.com/google/uuid"
)

type Request struct {
	URL         string        `json:"url"`
	Customshort string        `json:"customshort"`
	Expiry      time.Duration `json:"expiry"`
}

type Response struct {
	URL            string        `json:"url"`
	Custom         string        `json:"custom"`
	Expiry         time.Duration `json:"expiry"`
	XRateRemaining int           `json:"xrateremaining"`
	XRateLimitRest time.Duration `json:"xratelimitrest"`
}

func ShoternUrl(c *fiber.Ctx) error {

	body := new(Request)
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot Parse JSON"})

	}
	// implementing rate limit
	db2 := database.CreateClient(1)
	defer db2.Close()

	ipValue, err := db2.Get(database.Ctx, c.IP()).Result()

	if err == redis.Nil {
		_ = db2.Set(database.Ctx, c.IP(), os.Getenv("API_QUOTA"), 30*60*time.Second).Err()
	} else {
		intIpValue, _ := strconv.Atoi(ipValue)

		if intIpValue <= 0 {
			limit, _ := db2.TTL(database.Ctx, c.IP()).Result()
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error":           "Rate Limit Exceeded",
				"rate_limit_rest": limit / time.Nanosecond / time.Minute,
			})
		}

	}

	// check if url is valid url
	if !govalidator.IsURL(body.URL) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid URL"})
	}

	// check for domain err

	if !helpers.RemoveDomainError(body.URL) {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Invalid URL"})
	}

	body.URL = helpers.EnforceHttp(body.URL)

	var id string
	if body.Customshort == "" {
		id = uuid.New().String()[0:6]
	} else {
		id = body.Customshort
	}

	db3 := database.CreateClient(0)
	db3.Close()

	idVal, _ := db3.Get(database.Ctx, id).Result()
	if idVal != "" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"errors": "URL custom is already in use",
		})
	}

	if body.Expiry == 0 {
		body.Expiry = 24
	}

	err = db3.Set(database.Ctx, id, body.URL, body.Expiry*3600*time.Second).Err()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"errors": "Unable to connect server",
		})
	}

	db2.Decr(database.Ctx, c.IP())

	resp := Response{
		URL:            body.URL,
		Custom:         "",
		Expiry:         body.Expiry,
		XRateRemaining: 10,
		XRateLimitRest: 30,
	}
	rateVal, _ := db2.Get(database.Ctx, c.IP()).Result()
	resp.XRateRemaining, _ = strconv.Atoi(rateVal)

	ttl, _ := db2.TTL(database.Ctx, c.IP()).Result()

	resp.XRateLimitRest = ttl / time.Nanosecond / time.Minute

	resp.Custom = os.Getenv("DOMAIN") + "/" + id

	return c.Status(fiber.StatusOK).JSON(resp)

}
