## Instructions
- When running code, always use `wails dev` for development and `wails build` for production builds.
- Before running `wails dev` check to make sure that there isn't already an instance running.
- If you are ever extending or adding new functionality in terms of interacting with the Fabric REST APIs, make sure to use the `microsoftdocs/mcp` and `upstash/context-7` MCP servers. 
- When performing calculations, always prefer to compute them with DuckDB SQL queries rather than in Go code for better performance and consistency. If you want to deviate from this, you will need explicit approval from the user and an explaination as to why.