# Pull Pal

A digital assistant that writes code and collaborates with humans on git repositories.

This tool is based on a previous [experiment/proof of concept](https://github.com/mobyvb/mobys-gpt-app).

## Table of Contents

- [Configuration](#configuration)
- [Running](#running)
- [Usage](#usage)
- [Contributing](#contributing)
- [License](#license)


## Configuration

The easiest way to configure the tool is by adding a config yaml file. The default filepath is `~/.pull-pal.yaml`, but you can pass in any filepath by using the `--config` argument.

The minimal configuration you need looks like this:

```
handle: [username of your bot's Github account]
email: [email of your bot's Github account]
repos: [list of repositories, e.g. "github.com/owner/name" that bot will monitor]
users-to-listen-to: [list of Github users who your bot will interact with on Github issues and PRs]
required-issue-labels: [list of issue labels that an issue must have in order to be considered by the bot (can be empty)]
github-token: ghp_xxx
open-ai-token: sk-xxx
```

You can acquire the Github token under "developer settings" in Github, in the "personal access tokens" section. The necessary requirements are:
* this token must be created for the same account associated with `handle` and `email` in your config. Important to remember if you are using a separate Github account for your bot
* the token must have permissions to interact with the repositories configured in `repos`
* the token must have read and write permission to commit statuses, repository contents, discussions, issues, and pull requests

You can generate an API key for OpenAI by logging in to platform.openai.com, then going to https://platform.openai.com/account/api-keys
* If you do not have GPT4 access, you may need to switch GPT4 for GPT3.5Turbo in ./llm/openai.go (todo make configurable)

You can use your own `handle` and `email` in the configuration, but I prefer to use a separate Github account so that it is clear what changes come from me vs. the bot.

## Running

To run, all you need to do is execute 
```
go run main.go
```

To provide a custom config file, execute 
```
go run main.go --config=/path/to/config.yaml
```

To provide specific configs outside of a config file, execute 
```
go run main.go --handle mybothandle --email mybotemail@mail.test etc...
```

## Usage

Once Pull Pal is running with your config, you should be able to create issues in your repository for the bot to respond to.

Be clear, specific, and detailed in the description of what you want done. Mention specific files and technical details that you might be able to provide at the time of writing. This will minimize the number of iterations necessary to get good code, or manual intervention to fix broken code.

Currently, you are required to mention a list of files that will need to be added, modified, or read (i.e. for additional context), in a comma-separated list at the end of the issue body.

Example of an issue body that should be parseable by Pull Pal:

```
Add an index.html file. It should have a content section populated with a heading and body about a cool new product. The content section should be centered horizontally and vertically, have a border radius, and have a drop shadow. It should have an off-white background color. The body of the page should have a soft, light color that is not white. The page should use a sans-serif font. The heading should be a different color than the rest of the text.

Add a main.go file that serves index.html on port 8080.

---

Files: main.go, index.html
```

After creating your first issue, with an account configured in the `users-to-listen-to` list, add the `required-issue-labels`, if any, and your Pull Pal should notice it and begin working on it shortly. If any errors occur, the best place to look is in your Pull Pal logs. If you are still having an issue or if you have any suggestions, please [open an issue](https://github.com/mobyvb/pull-pal/issues/new).

## Contributing

I encourage contributing directly to this repository, or forking it and using it to accomplish other goals. If you would like to work together on this project, please contact me via email at mobyvb@gmail.com.

## License

The [license](./LICENSE) is GPL 3.0, which basically means that you can freely use/distribute/modify the code here, as long as you maintain the GPL license. You can build a product out of this tool and even try to make money from it. However, you cannot close-source code that you derived from this project. Keep it open for others to use.
