import { Meta, Story } from "@storybook/vue3";
import TableComponent from "./Table.vue";
import TableRowComponent from "./TableRow.vue";

export default {
  title: "Components/Table",
  component: TableComponent
} as Meta<typeof TableComponent>;

const TableTemplate: Story = (args) => ({
  components: { TableComponent, TableRowComponent },
  setup() {
    return {
      args,
    };
  },
  template: `<TableComponent>
    <TableRowComponent title="Row 1">
      This is a row 1
    </TableRowComponent>
    <TableRowComponent title="Row 2">
      This is a row 2
    </TableRowComponent>
    <TableRowComponent title="Row 3">
      This is a row 3
    </TableRowComponent>
  </TableComponent>`,
});

export const Table = TableTemplate.bind({});
Table.args = {};
