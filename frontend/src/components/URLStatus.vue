<template>
  <div class="hello">
    <h2>Status</h2>
    <div v-if="status == null">
      Loading....
    </div>
    <div v-else-if="status == false">
      Looks like there's nothing on this domain.
    </div>
    <div v-else>
      <div v-if="status.Blacklisted">
        {{ this.$route.params.url }} is blacklisted. You cannot view this page.
      </div>
      <div v-else-if="status.Cooldown">
        {{ this.$route.params.url }} is on a cooldown period. You can visit it again in {{ status.Cooldown }} seconds.
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'URLStatus',
  data () {
    return {
      status: null
    }
  },
  created() {
    this.fetchData();
  },
  methods: {
    fetchData () {
      fetch(`http://proxy.cop/api/url/${this.$route.params.url}/status`)
        .then(result => {
          return result.json();
        })
        .then(result => {
          this.status = result || false;
          if (this.status.Cooldown) {
            setTimeout(this.tickCooldown, 1000);
          }
        });
    },
    tickCooldown() {
      console.log("Status", this.status);
      this.$set(this.status, "Cooldown", this.status.Cooldown - 1);
      if (this.status.Cooldown > 0) {
        setTimeout(this.tickCooldown, 1000);
      }
    }
  }
}
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped>
h1, h2 {
  font-weight: normal;
}
ul {
  list-style-type: none;
  padding: 0;
}
li {
  display: inline-block;
  margin: 0 10px;
}
a {
  color: #42b983;
}
</style>
