package routes

import (
	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
	"github.com/gondsuryaprakash/shortenurl/database"
)

func ResoleUrl(c *fiber.Ctx) error {

	url := c.Params("url")

	db := database.CreateClient(0)
	defer db.Close()

	value, err := db.Get(database.Ctx, url).Result()
	if err == redis.Nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "url not found in database"})
	} else if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Not able to connect with DB"})
	}

	db1 := database.CreateClient(1)
	defer db1.Close()

	_ = db1.Incr(database.Ctx, "counter")

	return c.Redirect(value, 301)

}
