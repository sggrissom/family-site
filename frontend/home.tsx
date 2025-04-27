import * as preact from "preact"
import * as rpc from "vlens/rpc";

type Data = {}

export async function fetch(route: string, prefix: string) {
    return rpc.ok<Data>({})
}

export function view(route: string, prefix: string, data: Data): preact.ComponentChild {
    return <HeroSection/>
}

const HeroSection = () => {
  return (
    <div className="hero">
      <h1>Welcome to a Family Site</h1>
      <p style={{ marginBottom: '40px' }}>Track some family stuff.</p>
      <a className="cta-button" href="/register">Get Started</a>
      <a className="cta-button" href="/explore">Explore</a>
    </div>
  );
};
