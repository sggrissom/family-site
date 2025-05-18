import * as preact from "preact"
import { Footer, Header } from "home"
import * as server from "@app/server"
import * as rpc from "vlens/rpc"
import * as core from "vlens/core"
import { getAuth } from "util/authCache"

export async function fetch(route: string, prefix: string) {
    return rpc.ok<server.Empty>({})
}

export function view(route: string, prefix: string, data: server.Empty): preact.ComponentChild {
    const auth = getAuth()
    if (!(auth && auth.Id > 0)) {
        core.setRoute("/")
    }
    return <>
        <Header/>
        <div className="container family-dashboard">
            <h2>Family Dashboard</h2>
            <p>Welcome family! Here’s an overview of your family.</p>
            <h3>Family Members</h3>
            <div className="actions">
                <a className="button" href="/children/add">
                    Add Person
                </a>
                <a className="button" href="/milestones/add">
                    Add Milestone
                </a>
                <a className="button" href={`/family/edit/1`}>
                    Edit Family
                </a>
            </div>
        </div>
        <Footer />
    </>
}
