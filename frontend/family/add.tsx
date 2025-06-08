import * as preact from "preact";
import { Footer, Header } from "home";
import * as server from "@app/server";
import * as vlens from "vlens";
import * as rpc from "vlens/rpc";
import * as core from "vlens/core";
import { getAuth } from "util/authCache";
import { FunctionalComponent } from "preact";

type Form = {
  name: string;
  description: string;
  id: number;
  error: string;
};

const useForm = vlens.declareHook(
  (): Form => ({
    name: "",
    description: "",
    id: 0,
    error: "",
  }),
);

export async function fetch(route: string, prefix: string) {
  return rpc.ok<server.Empty>({});
}

export function view(
  route: string,
  prefix: string,
  data: server.Empty,
): preact.ComponentChild {
  const auth = getAuth();
  if (!(auth && auth.Id > 0)) {
    core.setRoute("/");
  }
  const form = useForm();
  return (
    <>
      <Header />
      <div className="container family-dashboard">
        <h2>Add a Family</h2>
        <AddFamilyForm form={form} />
      </div>
      <Footer />
    </>
  );
}

interface AddFamilyFormProps {
  form: Form;
}

const AddFamilyForm: FunctionalComponent<AddFamilyFormProps> = ({ form }) => {
  return (
    <form method="post" action="/family/add">
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
        <label htmlFor="name">Description:</label>
        <input
          type="text"
          id="description"
          name="description"
          {...vlens.attrsBindInput(vlens.ref(form, "description"))}
        />
      </div>

      <input type="hidden" name="id" defaultValue={form.id} />

      <button onClick={vlens.cachePartial(onAddFamilyClicked, form)}>
        Submit Family
      </button>
      <a href="/" className="button button-secondary">
        Cancel
      </a>
    </form>
  );
};

async function onAddFamilyClicked(form: Form, event: Event) {
  event.preventDefault();
  server.AddFamily({
    Id: form.id,
    Name: form.name,
    Description: form.description,
  });
  core.setRoute("/");
  vlens.scheduleRedraw();
}

