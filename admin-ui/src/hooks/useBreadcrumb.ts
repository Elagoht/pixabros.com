import { useLayoutEffect } from "react";
import {
  type BreadcrumbItem,
  useBreadcrumbStore,
} from "@/lib/stores/breadcrumb";

const useBreadcrumb = (items: BreadcrumbItem[]) => {
  const setItems = useBreadcrumbStore((s) => s.setItems);
  const serialized = JSON.stringify(items);

  useLayoutEffect(() => {
    setItems(JSON.parse(serialized) as BreadcrumbItem[]);
    return () => setItems([]);
  }, [serialized, setItems]);
};

export default useBreadcrumb;
