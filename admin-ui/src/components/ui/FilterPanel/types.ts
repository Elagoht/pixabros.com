export interface FilterOption {
  label: string;
  value: string;
}

export type FilterDef =
  | {
      type: "text";
      key: string;
      label: string;
      placeholder?: string;
    }
  | {
      type: "date";
      key: string;
      label: string;
    }
  | {
      type: "select";
      key: string;
      label: string;
      placeholder?: string;
      disabled?: boolean;
      options: FilterOption[];
    }
  | {
      type: "multiselect";
      key: string;
      label: string;
      options: FilterOption[];
    }
  | {
      type: "number-range";
      key: string;
      label: string;
      min?: number;
      max?: number;
      step?: number;
    };
