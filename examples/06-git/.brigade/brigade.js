const { events, Job } = require("@brigadecore/brigadier");
        events.on("brigade.sh/cli", "exec", async event => {
          let job1 = new Job("my-first-job", "debian:latest", event);
          job1.primaryContainer.command = ["echo"];
          job1.primaryContainer.arguments = ["My first job!"];
          await job1.run();
                
          let job2 = new Job("dind", "docker:stable-dind", event);
          job.primaryContainer.environment.DOCKER_HOST = "localhost:2375";
          job.primaryContainer.command = ["sh"];
          job.primaryContainer.arguments = [
               "-c",
               // Wait for the Docker daemon to start up
               // And then pull the image
               "sleep 20 && docker pull busybox"
               ];
          // Run the Docker daemon in a sidecar container
          job.sidecarContainers = {
             "docker": new Container("docker:stable-dind")
          };
          job.sidecarContainers.docker.privileged = true
          await job2.run();
        });
        events.process();
