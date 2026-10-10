export type MeEnterpriseInquiry = {
  id: string;
  company_name?: string;
  estimated_seats?: number;
  offered_seat_quantity?: number;
  message?: string;
  status: string;
  status_label?: string;
  created_at?: string;
  updated_at?: string;
  can_checkout?: boolean;
};

export type MeEnterpriseInquiriesChrome = {
  title?: string;
  empty?: string;
  col_company?: string;
  col_status?: string;
  col_requested?: string;
  col_offered?: string;
  col_created?: string;
  col_updated?: string;
  col_message?: string;
  pay_label?: string;
  pay_pending_label?: string;
  view_label?: string;
  details_label?: string;
  filter_status?: string;
  filter_status_desc?: string;
  filter_all?: string;
  filter_search?: string;
  search_placeholder?: string;
  search_description?: string;
  status_new?: string;
  status_contacted?: string;
  status_offered?: string;
  status_closed?: string;
  status_activated?: string;
  checkout_interval_required?: string;
  table_select_all?: string;
  table_select_row?: string;
  table_selected_fmt?: string;
  table_row_actions?: string;
  table_bulk_delete?: string;
  table_clear_selection?: string;
  delete_action_label?: string;
  delete_pending_label?: string;
  delete_confirm_message?: string;
  bulk_delete_confirm_message?: string;
};

export type MeEnterpriseInquiriesResponse = {
  items: MeEnterpriseInquiry[];
  meta?: {
    total?: number;
    skip?: number;
    limit?: number;
    page?: number;
    total_pages?: number;
    has_more?: boolean;
  };
  plan_id?: string;
  chrome?: MeEnterpriseInquiriesChrome;
  q?: string;
};
