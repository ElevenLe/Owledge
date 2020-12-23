import React, { Component } from 'react';
import TopBar from "./TopBar"
import Main from "./Main"
import './../styles/App.css';
import { Layout } from 'antd';
const { Header, Content } = Layout;



class App extends Component{
  render() {
    return (
      <div className="App">
        <Layout>
        <Header className="top-header">
        <TopBar />
        </Header>
        <Content>
        <Main />
        </Content>
        </Layout>
      </div>
    )
  }

// handleLoginSuccessd = (token) => {
//   console.log(token)
//     localStorage.setItem(TOKEN_KEY, token)
//     this.setState({
//       isLoggedIn: true
//     })
// }

// handleLogout = () => {
//   console.log("logout")
//   localStorage.removeItem(TOKEN_KEY)
//   this.setState({
//     isLoggedIn: false
//   })
// }
}


export default App;
