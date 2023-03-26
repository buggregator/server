import {Meta, Story} from "@storybook/vue3";
import EmailAddressesComponent from './EmailAddresses.vue';

export default {
    title: "Pages/Smtp/Parts",
    component: EmailAddressesComponent
} as Meta<typeof EmailAddressesComponent>;

const Template: Story = (args) => ({
    components: {EmailAddressesComponent},
    setup() {
        return {
            args,
        };
    },
    template: `
      <EmailAddressesComponent v-bind="args"/>`,
});

export const EmailAddresses = Template.bind({});

EmailAddresses.args = {
    addresses: [
        {
            name: 'John Doe',
            email: 'john-doe@example.com',
        },
        {
            name: 'Jane Smith',
            email: 'JaneSmith@example.com',
        },
        {
            email: 'saraConor@example.com',
        }
    ],
};