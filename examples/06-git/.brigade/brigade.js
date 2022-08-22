const { events, Job } = require("@brigadecore/brigadier");
        events.on("brigade.sh/cli", "exec", async event => {
          let job1 = new Job("my-first-job", "debian:latest", event);
          job1.primaryContainer.command = ["echo"];
          job1.primaryContainer.arguments = ["My first job!"];
          await job1.run();
          let job2 = new Job("my-second-job", "debian:latest", event);
          job2.primaryContainer.command = ["echo"];
          job2.primaryContainer.arguments = ["My second job!"];
          await job2.run();
        });
        events.process();
