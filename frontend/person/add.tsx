import * as preact from "preact"
import { Footer, Header } from "home"
import * as server from "@app/server"
import * as vlens from "vlens";
import * as rpc from "vlens/rpc"
import * as core from "vlens/core"
import { getAuth } from "util/authCache"
import { FunctionalComponent } from "preact";

type Form = {
    personType: number
    birthdate: string
    name: string
    gender: number
    id: number
    error: string
}

const useForm = vlens.declareHook((): Form => ({
    personType: 0, birthdate: "", name:"", gender: 0, id: 0, error: ""
}))

export async function fetch(route: string, prefix: string) {
    return rpc.ok<server.Empty>({})
}

export function view(route: string, prefix: string, data: server.Empty): preact.ComponentChild {
    const auth = getAuth()
    if (!(auth && auth.Id > 0)) {
        core.setRoute("/")
    }
    const form = useForm()
    return <>
        <Header/>
        <div className="container family-dashboard">
            <h2>Add a Person</h2>
            <AddPersonForm form={form} />
        </div>
        <Footer />
    </>
}

interface AddPersonFormProps {
  form: Form;
}

const AddPersonForm: FunctionalComponent<AddPersonFormProps> = ({ form }) => {
  return (
    <form method="post" action="/children/add">
      <div className="form-group">
        <label htmlFor="personType">Person Type:</label>
        <select id="personType" name="personType" 
        {...vlens.attrsBindInput(vlens.ref(form, "personType"))}>
          <option value="1">Parent</option>
          <option value="2">Child</option>
        </select>
      </div>

      <div className="form-group">
        <label htmlFor="birthdate">Birthday:</label>
        <input
          type="date"
          id="birthdate"
          name="birthdate"
          {...vlens.attrsBindInput(vlens.ref(form, "birthdate"))}
        />
      </div>

      <div className="form-group">
        <label htmlFor="name">Name:</label>
        <input
          type="text"
          id="name"
          name="name"
          {...vlens.attrsBindInput(vlens.ref(form, "name"))}
        />
      </div>

      <div className="form-group">
        <label htmlFor="gender">Gender:</label>
        <select id="gender" name="gender"
          {...vlens.attrsBindInput(vlens.ref(form, "gender"))}>
          <option value="1">Male</option>
          <option value="2">Female</option>
        </select>
      </div>

      <input type="hidden" name="id" defaultValue={form.id} />

      <button onClick={vlens.cachePartial(onAddPersonClicked, form)}>Submit Person</button>
      <a href="/" className="button button-secondary">Cancel</a>
    </form>
  );
};

async function onAddPersonClicked(form: Form, event: Event) {
    event.preventDefault()
    server.AddPerson({
      Id: form.id,
      PersonType: form.personType,
      Gender: form.gender,
      Birthdate: form.birthdate,
      Name: form.name,
    })
    vlens.scheduleRedraw()
}