import { useAuth } from "./auth";
import "./App.css";
import SignupPage from "./pages/Signup";
import ProductsPage from "./pages/Products";
import ProductDetailPage from "./pages/ProductDetail";
import LoginPage from "./pages/Login";
import CartPage from "./pages/Cart";
import AddressesPage from "./pages/Addresses";
import CheckoutPage from "./pages/Checkout";
import OrdersPage from "./pages/Orders";
import OrderDetailPage from "./pages/OrderDetail";
import { Navigate } from "react-router-dom";

import { Routes } from "react-router-dom";
import { Route } from "react-router-dom";
import NavBar from "./components/NavBar";

function RequireAuth(props: { children: React.ReactNode }) {
  const { accessToken } = useAuth();
  if (!accessToken) return <Navigate to="/login" replace />;
  return <>{props.children}</>;
}

export default function App() {
  return (
    <div>
      <NavBar />
      <div style={{ padding: 12 }}>
        <Routes>
          <Route path="/signup" element={<SignupPage />} />
          <Route path="/" element={<ProductsPage />} />
          <Route path="/products/:id" element={<ProductDetailPage />} />

          <Route path="/login" element={<LoginPage />} />

          <Route
            path="/cart"
            element={
              <RequireAuth>
                <CartPage />
              </RequireAuth>
            }
          />
          <Route
            path="/addresses"
            element={
              <RequireAuth>
                <AddressesPage />
              </RequireAuth>
            }
          />
          <Route
            path="/checkout"
            element={
              <RequireAuth>
                <CheckoutPage />
              </RequireAuth>
            }
          />
          <Route
            path="/orders"
            element={
              <RequireAuth>
                <OrdersPage />
              </RequireAuth>
            }
          />
          <Route
            path="/orders/:id"
            element={
              <RequireAuth>
                <OrderDetailPage />
              </RequireAuth>
            }
          />

          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </div>
    </div>
  );
}
