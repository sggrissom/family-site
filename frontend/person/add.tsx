import * as preact from "preact"
import { Footer, Header } from "home"
import * as server from "@app/server"
import * as vlens from "vlens";
import * as rpc from "vlens/rpc"
import * as core from "vlens/core"
import { getAuth } from "util/authCache"
import { FunctionalComponent } from "preact";

type Form = {
    personType: string
    birthdate: string
    name: string
    gender: string
    id: string
    error: string
}

const useForm = vlens.declareHook((): Form => ({
    personType: "", birthdate: "", name:"", gender: "", id: "", error: ""
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
          <option value="parent">Parent</option>
          <option value="child">Child</option>
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
          <option value="male">Male</option>
          <option value="female">Female</option>
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
    console.log(form)
    vlens.scheduleRedraw()
}