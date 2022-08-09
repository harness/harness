import React, { useRef, useState, useCallback } from "react";
import { useHistory } from 'react-router-dom'
import styles from "./Register.module.scss";
import { useOnRegister } from 'services/pm'
import routes from 'RouteDefinitions'

import Link from "../../components/Link/Link";
import Input from "../../components/Input/input";
import Button from "../../components/Button/button";
import logo from "../../logo.svg"

// Renders the Register page.
export const Register = () => {
	const history = useHistory()
	const [error, setError] = useState(null);
	const [fullname, setFullname] = useState('')
	const [username, setUsername] = useState('')
	const [password, setPassword] = useState('')
	const { mutate } = useOnRegister({})

	const handleRegister = useCallback(() => {
		const formData = new FormData()
	
		formData.append("fullname", fullname);
		formData.append("password", password);
		formData.append("username", username);
	
		mutate(formData)
		  .then(() => {
			history.replace(routes.toLogin())
		  })
		  .catch(error => {
			// TODO: Use toaster to show error
			// eslint-disable-next-line no-console
			console.error({ error })
			setError(error);
		  })
	  }, [mutate, username, password, fullname, history])

	const alert =
		error && error.message ? (
			<div class="alert">{error.message}</div>
		) : undefined;

	return (
		<div className={styles.root}>
			<div className={styles.logo}>
				<img src={logo} />
			</div>
			<h2>Sign up for a new account</h2>
			{alert}
			<div className={styles.field}>
				<label>Full Name</label>
				<Input
					type="text"
					name="fullname"
					placeholder="Full Name"
					className={styles.input}
					onChange={e => setFullname(e.target.value)}
				/>
			</div>
			<div className={styles.field}>
				<label>Email</label>
				<Input
					type="text"
					name="username"
					placeholder="Email"
					className={styles.input}
					onChange={e => setUsername(e.target.value)}
				/>
			</div>
			<div className={styles.field}>
				<label>Password</label>
				<Input
					type="password"
					name="password"
					placeholder="Password"
					className={styles.input}
					onChange={e => setPassword(e.target.value)}
				/>
			</div>
			<div>
				<Button onClick={handleRegister} className={styles.submit}>
					Sign Up
				</Button>
			</div>
			<div className={styles.actions}>
				<span>
					Already have an account? <Link href="/login">Sign In</Link>
				</span>
			</div>
		</div>
	);
}