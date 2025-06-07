import * as preact from "preact";
import { Footer, Header } from "home";
import * as server from "@app/server";
import * as core from "vlens/core";
import { getAuth, clearAuth } from "util/authCache";
import { FunctionalComponent } from "preact";

export async function fetch(route: string, prefix: string) {
  return server.GetFamilyInfo({});
}

export function view(
  route: string,
  prefix: string,
  data: server.FamilyDataResponse,
): preact.ComponentChild {
  const auth = getAuth();
  if (!(auth && auth.Id > 0) || data.AuthUserId === 0) {
    clearAuth();
    core.setRoute("/");
    return;
  }
  if (!data.Family || data.Family.Id === 0) {
    core.setRoute("/family/add");
    return;
  }

  return (
    <>
      <Header />
      <div className="container family-dashboard">
        <h2>Family Dashboard</h2>
        <p>Welcome {data.Family.Name}! Here’s an overview of your family.</p>
        <h3>Family Members</h3>
        <PersonDashboard members={data.Members} />
        <div className="actions">
          <a className="button" href="/person/add">
            Add Person
          </a>
          <a className="button" href="/milestones/add">
            Add Milestone
          </a>
          <a className="button" href={`/family/add`}>
            Edit Family
          </a>
        </div>
      </div>
      <Footer />
    </>
  );
}

interface PersonDashboardProps {
  members: object[];
}

const PersonDashboard: FunctionalComponent<PersonDashboardProps> = ({
  members,
}) => {
  if (!members || members.length == 0) {
    return <p>no people</p>;
  }

  return members.map((person) => <p>{person.Name}</p>);
};
