import * as vlens from "vlens";
import * as server from "@app/server"

async function main() {
    vlens.initRoutes([
        vlens.routeHandler("/admin", () => import("@app/admin/admin")),
        vlens.routeHandler("/dashboard", () => import("@app/dashboard/dashboard")),
        vlens.routeHandler("/register", () => import("@app/account/register")),
        vlens.routeHandler("/login", () => import("@app/account/login")),
        vlens.routeHandler("/explore", () => import("@app/explore")),
        vlens.routeHandler("/", () => import("@app/home")),
    ]);
}

main();

(window as any).server = server 