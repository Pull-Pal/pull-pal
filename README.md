# Pull Pal

A digital assistant that writes code and collaborates with humans on git repositories.

This tool is based on a previous [experiment/proof of concept](https://github.com/mobyvb/mobys-gpt-app).

## Overview

The specific goals/functions of this tool may change over time, but the initial target functionality includes:

* "Bot" functionality (i.e. something that runs on a server somewhere and works mostly independently):
  * Monitoring open issues in a git repository, and automatically creating code changes (e.g. Github PRs) according to details in issue
  * Monitoring comments created on a code change, and updating the change accordingly
  * Integrates with OpenAI's GPT API (I'm still on the waitlist for this so may not be fully tested for a little bit)
* CLI functionality (i.e. something that accomplishes similar behavior to the bot functionality, but requires more human intervention):
  * Command to generate LLM prompt based on git repository issues (prompt can then be copied to a chatbot)
  * Command to generate a code change based on LLM response to prompt (e.g. output from above point copied to this command)

## Potential Evolution (future features following first foray)

* Gerrit support
  - including alternate sources for issue tracking + code changes
* Feedback/bug catching on code changes made by humans (add comments on changes)
* General LLM interface support - allow using local LLM, non-GPT APIs, etc...

## Contributing

I encourage contributing directly to this repository, or forking it and using it to accomplish other goals. If you would like to work together on this project, please contact me via email at mobyvb@gmail.com.

## License

The [license](./LICENSE) is GPL 3.0, which basically means that you can freely use/distribute/modify the code here, as long as you maintain the GPL 3.0 or a _more permissive_ license. You can build a product out of this tool and even try to make money from it. However, you cannot close-source code that you derived from this project. Keep it open for others to use. Don't be a jerk.
