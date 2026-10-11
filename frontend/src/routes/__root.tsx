import { useState } from 'react';
import { DashboardOutlined, TableOutlined, MenuFoldOutlined, MenuUnfoldOutlined } from '@ant-design/icons';
import { createRootRoute, Link, Outlet, useRouterState } from '@tanstack/react-router';
import { TanStackRouterDevtools } from '@tanstack/react-router-devtools';
import { Button, Layout, Menu, Tooltip } from 'antd';

const { Sider, Content, Footer } = Layout;
const CURRENT_YEAR = new Date().getFullYear();

export const Route = createRootRoute({ component: RootComponent });

function RootComponent() {
  const pathname = useRouterState({ select: state => state.location.pathname });
  const [collapsed, setCollapsed] = useState(true);

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider className="app-sidebar" width={240} collapsedWidth={64} collapsed={collapsed} trigger={null}>
        <div className={`app-sidebar-heading${collapsed ? ' is-collapsed' : ''}`}>
          {!collapsed && <Link className="app-brand" to="/">StarRocks Admin</Link>}
          <Tooltip title={collapsed ? 'Expand sidebar' : 'Collapse sidebar'} placement="right">
            <Button
              className="app-sidebar-toggle"
              type="text"
              aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
              aria-expanded={!collapsed}
              icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
              onClick={() => setCollapsed(value => !value)}
            />
          </Tooltip>
        </div>
        <nav aria-label="Main navigation">
          <Menu
            theme="dark"
            mode="inline"
            inlineCollapsed={collapsed}
            selectedKeys={[pathname.startsWith('/materialized-views') ? '/materialized-views' : pathname]}
            items={[
              {
                key: '/',
                icon: collapsed ? <Link to="/" aria-label="Dashboard"><DashboardOutlined /></Link> : <DashboardOutlined />,
                label: collapsed ? 'Dashboard' : <Link to="/">Dashboard</Link>,
                title: 'Dashboard',
              },
              {
                key: '/materialized-views',
                icon: collapsed ? <Link to="/materialized-views" aria-label="Materialized views"><TableOutlined /></Link> : <TableOutlined />,
                label: collapsed ? 'Materialized views' : <Link to="/materialized-views">Materialized views</Link>,
                title: 'Materialized views',
              },
            ]}
          />
        </nav>
      </Sider>
      <Layout style={{ minWidth: 0 }}>
        <Content className="app-content"><Outlet /></Content>
        <Footer className="app-footer">StarRocks Admin {CURRENT_YEAR}</Footer>
      </Layout>
      {import.meta.env.DEV && <TanStackRouterDevtools />}
    </Layout>
  );
}
