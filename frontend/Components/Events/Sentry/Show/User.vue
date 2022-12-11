<template>
    <section class="py-5 px-4 md:px-6 lg:px-8 border-b" v-if="event.user">
        <h3 class="text-muted font-bold uppercase text-sm mb-5">user</h3>

        <Table>
            <TableRow v-if="event.user.id" :title="'Id'">
                {{ event.user.id }}
            </TableRow>

            <TableRow v-if="event.user.username" :title="'User name'">
                {{ event.user.username }}
            </TableRow>

            <TableRow v-if="event.user.email" :title="'Email'">
                {{ event.user.email }}
            </TableRow>

            <TableRow v-if="event.user.ip_address" :title="'Ip address'">
                {{ event.user.ip_address }}
            </TableRow>
        </Table>
    </section>
</template>

<script>
import Table from "@/Components/UI/Table";
import TableRow from "@/Components/UI/TableRow";

export default {
    components: {
        Table, TableRow,
    },
    props: {
        event: Object
    }
}
</script>
