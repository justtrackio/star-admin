import { createRootRoute, Link, Outlet } from '@tanstack/react-router';
import { TanStackRouterDevtools } from '@tanstack/react-router-devtools';
import { Layout } from 'antd';

const { Header, Content, Footer } = Layout;
const CURRENT_YEAR = new Date().getFullYear();

export const Route = createRootRoute({ component: RootComponent });

function RootComponent() {
  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Header className="app-header"><Link to="/">StarRocks Admin</Link></Header>
      <Content className="app-content"><Outlet /></Content>
      <Footer className="app-footer">StarRocks Admin {CURRENT_YEAR}</Footer>
      {import.meta.env.DEV && <TanStackRouterDevtools />}
    </Layout>
  );
}
