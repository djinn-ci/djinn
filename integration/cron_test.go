package integration

import (
	"testing"

	"djinn-ci.com/integration/djinn"
)

func Test_CronCreate(t *testing.T) {
	cli, _ := djinn.NewClientWithLogger(tokens.get("gordon.freeman").Token, apiEndpoint, t)

	c, err := djinn.CreateCron(cli, djinn.CronParams{
		Name:     "Test_CronCreate",
		Schedule: djinn.Daily,
		Manifest: djinn.Manifest{
			Namespace: "",
			Driver: map[string]string{
				"type":      "docker",
				"image":     "golang",
				"workspace": "/go",
			},
		},
	})

	if err != nil {
		t.Fatal(err)
	}

	if err := c.Get(cli); err != nil {
		t.Fatal(err)
	}
}

func Test_CronUpdate(t *testing.T) {
	cli, _ := djinn.NewClientWithLogger(tokens.get("gordon.freeman").Token, apiEndpoint, t)

	c, err := djinn.CreateCron(cli, djinn.CronParams{
		Name:     "Test_CronUpdate",
		Schedule: djinn.Daily,
		Manifest: djinn.Manifest{
			Namespace: "",
			Driver: map[string]string{
				"type":      "docker",
				"image":     "golang",
				"workspace": "/go",
			},
		},
	})

	if err != nil {
		t.Fatal(err)
	}

	if err := c.Update(cli, djinn.CronParams{Schedule: djinn.Monthly}); err != nil {
		t.Fatal(err)
	}

	if c.Schedule != djinn.Monthly {
		t.Fatalf("unexpected cron schedule, expected=%q, got=%q\n", djinn.Monthly, c.Schedule)
	}
}

func Test_CronDelete(t *testing.T) {
	cli, _ := djinn.NewClientWithLogger(tokens.get("gordon.freeman").Token, apiEndpoint, t)

	c, err := djinn.CreateCron(cli, djinn.CronParams{
		Name:     "Test_CronDelete",
		Schedule: djinn.Daily,
		Manifest: djinn.Manifest{
			Namespace: "",
			Driver: map[string]string{
				"type":      "docker",
				"image":     "golang",
				"workspace": "/go",
			},
		},
	})

	if err != nil {
		t.Fatal(err)
	}

	if err := c.Delete(cli); err != nil {
		t.Fatal(err)
	}
}
