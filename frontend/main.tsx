import * as vlens from "vlens";
import * as server from "@app/server"

async function main() {
    vlens.initRoutes([
        vlens.routeHandler("/register", () => import("@app/account/register")),
        vlens.routeHandler("/explore", () => import("@app/explore")),
        vlens.routeHandler("/", () => import("@app/home")),
    ]);
}

main();

(window as any).server = server 