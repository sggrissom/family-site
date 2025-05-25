import * as preact from "preact"
import { Footer, Header } from "home"
import * as server from "@app/server"
import * as core from "vlens/core"
import { getAuth } from "util/authCache"
import { FunctionalComponent } from "preact"

export async function fetch(route: string, prefix: string) {
    return server.ListPeople({})
}

export function view(route: string, prefix: string, data: server.PersonListResponse): preact.ComponentChild {
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
            <PersonDashboard peopleNames={data.AllPersonNames} />
            <div className="actions">
                <a className="button" href="/person/add">
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

interface PersonDashboardProps {
    peopleNames: string[];
}

const PersonDashboard: FunctionalComponent<PersonDashboardProps> = ({ peopleNames }) => {
    if (!peopleNames || peopleNames.length == 0) {
        return <p>no people</p>
    }

    return peopleNames.map(personName => <p>{personName}</p>)
}
